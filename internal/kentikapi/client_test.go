package kentikapi_test

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	cloudexport "github.com/kentik/api-schema-public/gen/go/kentik/cloud_export/v202506"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi/kentikapitest"
)

func fastRetry(c *kentikapi.Config) {
	c.Retry = kentikapi.RetryConfig{MaxAttempts: 3, MinDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond}
}

func TestNew_RequiresCredentials(t *testing.T) {
	t.Parallel()
	_, err := kentikapi.New(t.Context(), kentikapi.Config{Email: "a@b.c"})
	require.ErrorContains(t, err, "email and token are required")
}

func TestNew_DefaultsToUSEndpoint(t *testing.T) {
	t.Parallel()
	c, err := kentikapi.New(t.Context(), kentikapi.Config{Email: "a@b.c", Token: "t"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	assert.Equal(t, kentikapi.EndpointUS, c.Target())
}

func TestNew_RegionEU(t *testing.T) {
	t.Parallel()
	c, err := kentikapi.New(t.Context(), kentikapi.Config{Email: "a@b.c", Token: "t", Region: kentikapi.RegionEU})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	assert.Equal(t, kentikapi.EndpointEU, c.Target())
}

func TestNew_UnknownRegion(t *testing.T) {
	t.Parallel()
	_, err := kentikapi.New(t.Context(), kentikapi.Config{Email: "a@b.c", Token: "t", Region: "APAC"})
	require.ErrorContains(t, err, "unknown Kentik region")
}

func TestClient_SendsAuthMetadataAndUserAgent(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()
	c := srv.Client(t, func(cfg *kentikapi.Config) { cfg.UserAgent = "terraform-provider-test/1.2.3" })

	_, err := c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "/kentik.cloud_export.v202506.CloudExportAdminService/ListCloudExports", calls[0].Method)
	assert.Equal(t, []string{"joe.doe@example.com"}, calls[0].Metadata.Get("x-ch-auth-email"))
	assert.Equal(t, []string{"dummy-token"}, calls[0].Metadata.Get("x-ch-auth-api-token"))
	require.NotEmpty(t, calls[0].Metadata.Get("user-agent"))
	assert.Contains(t, calls[0].Metadata.Get("user-agent")[0], "terraform-provider-test/1.2.3")
}

func TestClient_RetriesTransientCodes(t *testing.T) {
	t.Parallel()
	for _, code := range []codes.Code{codes.Unavailable, codes.ResourceExhausted} {
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			srv := kentikapitest.NewServer(t)
			fake := kentikapitest.NewCloudExportServer().Register(srv)
			srv.Start()
			c := srv.Client(t, fastRetry)

			fake.FailNext(2, code)
			_, err := c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
			require.NoError(t, err)
			assert.Len(t, srv.Calls(), 3)
		})
	}
}

func TestClient_GivesUpAfterMaxAttempts(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	fake := kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()
	c := srv.Client(t, fastRetry)

	fake.FailNext(10, codes.Unavailable)
	_, err := c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Len(t, srv.Calls(), 3)
}

func TestClient_DoesNotRetryOtherCodes(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()
	c := srv.Client(t, fastRetry)

	_, err := c.CloudExport.GetCloudExport(t.Context(), &cloudexport.GetCloudExportRequest{Id: "nope"})
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Len(t, srv.Calls(), 1)
}

func TestClient_RetryStopsWhenContextDone(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	fake := kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()
	c := srv.Client(t, func(cfg *kentikapi.Config) {
		cfg.Retry = kentikapi.RetryConfig{MaxAttempts: 100, MinDelay: time.Hour, MaxDelay: time.Hour}
	})

	fake.FailNext(100, codes.Unavailable)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.CloudExport.ListCloudExports(ctx, &cloudexport.ListCloudExportsRequest{})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.Len(t, srv.Calls(), 1)
}

func TestClient_PerAttemptTimeout(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	cloudexport.RegisterCloudExportAdminServiceServer(srv.Registrar(), slowServer{})
	srv.Start()
	c := srv.Client(t, func(cfg *kentikapi.Config) {
		cfg.Timeout = 20 * time.Millisecond
		cfg.Retry.MaxAttempts = 1
	})

	_, err := c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
}

func TestClient_LogFuncSeesOutcome(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()

	var mu sync.Mutex
	var methods []string
	var errs []error
	c := srv.Client(t, func(cfg *kentikapi.Config) {
		cfg.Log = func(_ context.Context, method string, _, _ any, err error) {
			mu.Lock()
			defer mu.Unlock()
			methods = append(methods, method)
			errs = append(errs, err)
		}
	})

	_, _ = c.CloudExport.GetCloudExport(t.Context(), &cloudexport.GetCloudExportRequest{Id: "x"})
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, methods, 1)
	assert.Contains(t, methods[0], "GetCloudExport")
	assert.Equal(t, codes.NotFound, status.Code(errs[0]))
}

func TestClient_RefusesTokenOverPlaintextUnlessInsecure(t *testing.T) {
	t.Parallel()
	srv := kentikapitest.NewServer(t)
	kentikapitest.NewCloudExportServer().Register(srv)
	srv.Start()

	// TLS transport against a plaintext server must fail, not leak the token.
	c := srv.Client(t, func(cfg *kentikapi.Config) {
		cfg.Insecure = false
		cfg.Retry.MaxAttempts = 1
		cfg.Timeout = time.Second
	})
	_, err := c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
	require.Error(t, err)
	assert.Empty(t, srv.Calls())
}

// TestClient_LoopbackTCP exercises a real TCP listener on 127.0.0.1 through the
// legacy apiurl mapping (http:// means plaintext), as used by older test setups.
func TestClient_LoopbackTCP(t *testing.T) {
	t.Parallel()
	lis, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := grpc.NewServer()
	cloudexport.RegisterCloudExportAdminServiceServer(s, kentikapitest.NewCloudExportServer())
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)

	endpoint, insecureTransport, err := kentikapi.EndpointFromAPIURL("http://" + lis.Addr().String())
	require.NoError(t, err)
	c, err := kentikapi.New(t.Context(), kentikapi.Config{
		Email: "a@b.c", Token: "t", Endpoint: endpoint, Insecure: insecureTransport,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	_, err = c.CloudExport.ListCloudExports(t.Context(), &cloudexport.ListCloudExportsRequest{})
	require.NoError(t, err)
}

type slowServer struct {
	cloudexport.UnimplementedCloudExportAdminServiceServer
}

func (slowServer) ListCloudExports(
	ctx context.Context, _ *cloudexport.ListCloudExportsRequest,
) (*cloudexport.ListCloudExportsResponse, error) {
	<-ctx.Done()
	return nil, status.FromContextError(ctx.Err()).Err()
}
