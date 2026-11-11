// Package kentikapi is a thin adapter over the generated gRPC clients in
// github.com/kentik/api-schema-public. It owns connection setup (TLS, per-RPC
// authentication, timeouts, retries, logging) so that Terraform resources only
// deal with generated request and response types.
//
// The package is deliberately service-agnostic: New returns a Client holding a
// single *grpc.ClientConn, and each Kentik service is exposed as a field built
// from that connection. Adding a service means adding one field and one line
// in New.
package kentikapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Well-known Kentik gRPC endpoints.
const (
	EndpointUS = "grpc.api.kentik.com:443"
	EndpointEU = "grpc.api.kentik.eu:443"
)

// Region selects a Kentik data centre.
type Region string

// Supported regions.
const (
	RegionUS Region = "US"
	RegionEU Region = "EU"
)

// Endpoint returns the gRPC endpoint for the region (US when empty).
func (r Region) Endpoint() (string, error) {
	switch r {
	case "", RegionUS:
		return EndpointUS, nil
	case RegionEU:
		return EndpointEU, nil
	default:
		return "", fmt.Errorf("unknown Kentik region %q (want %q or %q)", r, RegionUS, RegionEU)
	}
}

// DefaultTimeout bounds a single RPC attempt when Config.Timeout is zero.
const DefaultTimeout = 100 * time.Second

// LogFunc receives every RPC after it completes. resp is nil when err is not nil.
// Authentication data travels in metadata and is never passed to LogFunc.
type LogFunc func(ctx context.Context, method string, req, resp any, err error)

// Config configures a Client. Email and Token are required.
type Config struct {
	// Email and Token are sent as X-CH-Auth-Email and X-CH-Auth-API-Token metadata.
	Email string
	Token string
	// Endpoint is the gRPC target as host:port. Empty means the Region endpoint.
	Endpoint string
	// Region is used when Endpoint is empty. Empty means RegionUS.
	Region Region
	// UserAgent is prepended to the gRPC user agent.
	UserAgent string
	// Timeout bounds each RPC attempt. Zero means DefaultTimeout.
	Timeout time.Duration
	// Retry controls retries of transient failures.
	Retry RetryConfig
	// Insecure disables TLS. Only for local fakes and tests.
	Insecure bool
	// Log, if set, is called after every RPC attempt sequence completes.
	Log LogFunc
	// DialOptions are appended to the options built from the fields above
	// (for example a bufconn dialer in tests).
	DialOptions []grpc.DialOption
}

// Client gives access to Kentik gRPC services over one connection.
type Client struct {
	conn *grpc.ClientConn

	// CloudExport is the cloud_export v202506 admin service.
	CloudExport cloudexport.CloudExportAdminServiceClient
}

// New validates cfg and creates a Client. The connection is established lazily
// on the first RPC; ctx is reserved for future use (e.g. eager health checks).
func New(_ context.Context, cfg Config) (*Client, error) {
	if cfg.Email == "" || cfg.Token == "" {
		return nil, errors.New("kentikapi: email and token are required")
	}
	target := cfg.Endpoint
	if target == "" {
		ep, err := cfg.Region.Endpoint()
		if err != nil {
			return nil, err
		}
		target = ep
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	retry := cfg.Retry.withDefaults()

	transport := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})
	if cfg.Insecure {
		transport = insecure.NewCredentials()
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(transport),
		grpc.WithPerRPCCredentials(authCredentials{email: cfg.Email, token: cfg.Token, secure: !cfg.Insecure}),
		// Order: logging sees the final outcome, retry wraps per-attempt timeouts.
		grpc.WithChainUnaryInterceptor(
			logInterceptor(cfg.Log),
			retryInterceptor(retry),
			timeoutInterceptor(timeout),
		),
	}
	if cfg.UserAgent != "" {
		opts = append(opts, grpc.WithUserAgent(cfg.UserAgent))
	}
	opts = append(opts, cfg.DialOptions...)

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("kentikapi: create gRPC client for %q: %w", target, err)
	}
	return &Client{
		conn:        conn,
		CloudExport: cloudexport.NewCloudExportAdminServiceClient(conn),
	}, nil
}

// Conn returns the underlying connection, for building clients of services
// that do not have a field on Client yet.
func (c *Client) Conn() grpc.ClientConnInterface { return c.conn }

// Target returns the gRPC target the client connects to.
func (c *Client) Target() string { return c.conn.Target() }

// Close releases the connection.
func (c *Client) Close() error { return c.conn.Close() }

func timeoutInterceptor(d time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func logInterceptor(log LogFunc) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if log != nil {
			if err != nil {
				log(ctx, method, req, nil, err)
			} else {
				log(ctx, method, req, reply, nil)
			}
		}
		return err
	}
}
