package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
)

func TestClientConfig(t *testing.T) {
	// Not parallel: t.Setenv.
	tests := map[string]struct {
		raw      map[string]any
		env      map[string]string
		endpoint string
		region   kentikapi.Region
		insecure bool
	}{
		"defaults to US region": {raw: map[string]any{}, region: kentikapi.RegionUS},
		"region EU":             {raw: map[string]any{"region": "EU"}, region: kentikapi.RegionEU},
		"region from env":       {raw: map[string]any{}, env: map[string]string{"KTAPI_REGION": "eu"}, region: kentikapi.RegionEU},
		"endpoint wins":         {raw: map[string]any{"endpoint": "grpc.example.com:443", "apiurl": "https://api.kentik.eu", "region": "EU"}, endpoint: "grpc.example.com:443", region: kentikapi.RegionEU},
		"endpoint from env":     {raw: map[string]any{}, env: map[string]string{"KTAPI_ENDPOINT": "grpc.example.com:443"}, endpoint: "grpc.example.com:443", region: kentikapi.RegionUS},
		"apiurl https":          {raw: map[string]any{"apiurl": "https://api.kentik.eu"}, endpoint: "grpc.api.kentik.eu:443", region: kentikapi.RegionUS},
		"apiurl http from env":  {raw: map[string]any{}, env: map[string]string{"KTAPI_URL": "http://127.0.0.1:9955"}, endpoint: "127.0.0.1:9955", insecure: true, region: kentikapi.RegionUS},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for _, k := range []string{"KTAPI_REGION", "KTAPI_ENDPOINT", "KTAPI_URL"} {
				t.Setenv(k, tc.env[k])
			}
			raw := map[string]any{"email": "a@b.c", "token": "t"}
			for k, v := range tc.raw {
				raw[k] = v
			}
			d := schema.TestResourceDataRaw(t, providerSchema(), raw)
			cfg, err := clientConfig(t.Context(), d, "ua")
			require.NoError(t, err)
			assert.Equal(t, tc.endpoint, cfg.Endpoint)
			assert.Equal(t, tc.region, cfg.Region)
			assert.Equal(t, tc.insecure, cfg.Insecure)
			assert.Equal(t, "ua", cfg.UserAgent)
			assert.Nil(t, cfg.Log)
		})
	}
}

func TestGetRetryConfig(t *testing.T) {
	t.Parallel()
	d := schema.TestResourceDataRaw(t, providerSchema(), map[string]any{
		"email": "a@b.c", "token": "t",
		"retry": []any{map[string]any{"max_attempts": 7, "min_delay": "250ms", "max_delay": "2s"}},
	})
	r, err := getRetryConfig(t.Context(), d)
	require.NoError(t, err)
	// v0.4 ignored max_attempts because of a wrong type assertion; it is honoured now.
	assert.Equal(t, kentikapi.RetryConfig{MaxAttempts: 7, MinDelay: 250 * time.Millisecond, MaxDelay: 2 * time.Second}, r)

	d = schema.TestResourceDataRaw(t, providerSchema(), map[string]any{"email": "a@b.c", "token": "t"})
	r, err = getRetryConfig(t.Context(), d)
	require.NoError(t, err)
	assert.Equal(t, kentikapi.RetryConfig{MaxAttempts: 100, MinDelay: time.Second, MaxDelay: 5 * time.Minute}, r)
}
