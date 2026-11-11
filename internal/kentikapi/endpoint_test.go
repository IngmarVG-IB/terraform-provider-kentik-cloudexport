package kentikapi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
)

func TestEndpointFromAPIURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in       string
		endpoint string
		insecure bool
		err      string
	}{
		{in: "https://api.kentik.com", endpoint: "grpc.api.kentik.com:443"},
		{in: "https://api.kentik.eu", endpoint: "grpc.api.kentik.eu:443"},
		{in: "https://api.example.com:8443/", endpoint: "grpc.api.example.com:8443"},
		{in: "http://127.0.0.1:9955", endpoint: "127.0.0.1:9955", insecure: true},
		{in: "http://[::1]:9955", endpoint: "[::1]:9955", insecure: true},
		{in: "http://fake-api", endpoint: "grpc.fake-api:80", insecure: true},
		{in: "ftp://api.kentik.com", err: "scheme must be http or https"},
		{in: "api.kentik.com", err: "scheme must be http or https"},
		{in: "https://", err: "has no host"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, insecure, err := kentikapi.EndpointFromAPIURL(tc.in)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.endpoint, got)
			assert.Equal(t, tc.insecure, insecure)
		})
	}
}

func TestRegionEndpoint(t *testing.T) {
	t.Parallel()
	for region, want := range map[kentikapi.Region]string{
		"":                 kentikapi.EndpointUS,
		kentikapi.RegionUS: kentikapi.EndpointUS,
		kentikapi.RegionEU: kentikapi.EndpointEU,
	} {
		got, err := region.Endpoint()
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestRetryBackoffBounds(t *testing.T) {
	t.Parallel()
	r := kentikapi.RetryConfig{MinDelay: 10 * time.Millisecond, MaxDelay: 80 * time.Millisecond}
	for n := uint(1); n <= 10; n++ {
		for range 50 {
			d := kentikapi.Backoff(r, n)
			assert.GreaterOrEqual(t, d, r.MinDelay)
			assert.LessOrEqual(t, d, r.MaxDelay)
		}
	}
	// First retry never waits more than MinDelay.
	assert.Equal(t, r.MinDelay, kentikapi.Backoff(r, 1))
}
