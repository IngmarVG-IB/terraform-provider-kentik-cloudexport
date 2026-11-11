// Package kentikapitest provides in-process fake Kentik gRPC servers for
// offline tests. Servers listen on a bufconn pipe, so no network is used.
//
// To fake another service, implement its generated *Server interface (embed the
// generated Unimplemented*Server) and register it with Server.Register before
// calling Start.
package kentikapitest

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
)

const bufSize = 1 << 20

// Target is the dummy gRPC target used with the bufconn dialer.
const Target = "passthrough:///bufnet"

// Call records one RPC received by the fake server.
type Call struct {
	Method   string
	Metadata metadata.MD
}

// Server is a bufconn-backed gRPC server.
type Server struct {
	t        testing.TB
	listener *bufconn.Listener
	grpc     *grpc.Server

	mu    sync.Mutex
	calls []Call
}

// NewServer creates a server. Register services, then call Start. The server is
// stopped automatically when the test ends.
func NewServer(t testing.TB, opts ...grpc.ServerOption) *Server {
	t.Helper()
	s := &Server{t: t, listener: bufconn.Listen(bufSize)}
	opts = append([]grpc.ServerOption{grpc.ChainUnaryInterceptor(s.record)}, opts...)
	s.grpc = grpc.NewServer(opts...)
	return s
}

// Register exposes the underlying registrar, e.g.
// cloudexport.RegisterCloudExportAdminServiceServer(s.Registrar(), impl).
func (s *Server) Registrar() grpc.ServiceRegistrar { return s.grpc }

// Start serves in the background until the test ends.
func (s *Server) Start() *Server {
	go func() {
		if err := s.grpc.Serve(s.listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			s.t.Errorf("fake gRPC server: %v", err)
		}
	}()
	s.t.Cleanup(s.grpc.Stop)
	return s
}

// DialOptions returns the options that route a client to this server.
func (s *Server) DialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return s.listener.DialContext(ctx)
		}),
	}
}

// Configure points a kentikapi.Config at this server (plaintext over bufconn).
func (s *Server) Configure(cfg *kentikapi.Config) {
	cfg.Endpoint = Target
	cfg.Insecure = true
	cfg.DialOptions = append(cfg.DialOptions, s.DialOptions()...)
}

// Calls returns a copy of the RPCs received so far.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.calls...)
}

func (s *Server) record(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.mu.Lock()
	s.calls = append(s.calls, Call{Method: info.FullMethod, Metadata: md.Copy()})
	s.mu.Unlock()
	return handler(ctx, req)
}

// Client is a convenience that builds a kentikapi.Client connected to s with
// dummy credentials. The client is closed when the test ends.
func (s *Server) Client(t testing.TB, mutate ...func(*kentikapi.Config)) *kentikapi.Client {
	t.Helper()
	cfg := kentikapi.Config{
		Email: "joe.doe@example.com",
		Token: "dummy-token",
	}
	s.Configure(&cfg)
	for _, m := range mutate {
		m(&cfg)
	}
	c, err := kentikapi.New(t.Context(), cfg)
	if err != nil {
		t.Fatalf("kentikapi.New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
