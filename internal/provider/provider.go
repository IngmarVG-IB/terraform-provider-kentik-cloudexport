package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/kentikapi"
)

const (
	apiURLKey      = "apiurl"
	endpointKey    = "endpoint"
	regionKey      = "region"
	emailKey       = "email"
	tokenKey       = "token"
	retryKey       = "retry"
	maxAttemptsKey = "max_attempts"
	minDelayKey    = "min_delay"
	maxDelayKey    = "max_delay"
	logPayloadsKey = "log_payloads"

	// Defaults documented since v0.3; kept for compatibility.
	defaultMaxAttempts = 100
	defaultMinDelay    = "1s"
	defaultMaxDelay    = "5m"

	providerName = "terraform-provider-kentik-cloudexport"
)

// Version is the provider version, set by main from the release ldflags.
var Version = "dev" //nolint:gochecknoglobals

// New returns the Cloud Export provider.
func New() *schema.Provider {
	return NewWithClientConfig(nil)
}

// NewWithClientConfig returns the provider; mutate, if not nil, may adjust the
// API client configuration after it is built from the provider settings. Tests
// use it to route the client to an in-process fake server.
func NewWithClientConfig(mutate func(*kentikapi.Config)) *schema.Provider {
	p := &schema.Provider{
		Schema: providerSchema(),
		ResourcesMap: map[string]*schema.Resource{
			"kentik-cloudexport_item": resourceCloudExport(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"kentik-cloudexport_list": dataSourceCloudExportList(),
			"kentik-cloudexport_item": dataSourceCloudExportItem(),
		},
	}
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (any, diag.Diagnostics) {
		cfg, err := clientConfig(ctx, d, p.UserAgent(providerName, Version))
		if err != nil {
			return nil, diag.FromErr(err)
		}
		if mutate != nil {
			mutate(&cfg)
		}
		tflog.Debug(ctx, "Creating Kentik API client", map[string]any{
			"endpoint": cfg.Endpoint, "region": string(cfg.Region), "email": cfg.Email,
			"max_attempts": cfg.Retry.MaxAttempts, "min_delay": cfg.Retry.MinDelay.String(),
			"max_delay": cfg.Retry.MaxDelay.String(),
		})
		client, err := kentikapi.New(ctx, cfg)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return client, nil
	}
	return p
}

func providerSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		emailKey: {
			Type:        schema.TypeString,
			Required:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_AUTH_EMAIL", nil),
			Description: "Authorization email (required). Can also be specified with KTAPI_AUTH_EMAIL environment variable.",
		},
		tokenKey: {
			Type:        schema.TypeString,
			Sensitive:   true,
			Required:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_AUTH_TOKEN", nil),
			Description: "Authorization token (required). Can also be specified with KTAPI_AUTH_TOKEN environment variable.",
		},
		regionKey: {
			Type:     schema.TypeString,
			Optional: true,
			DefaultFunc: func() (any, error) {
				return strings.ToUpper(envOr("KTAPI_REGION", string(kentikapi.RegionUS))), nil
			},
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(
				[]string{string(kentikapi.RegionUS), string(kentikapi.RegionEU)}, true)),
			Description: "Kentik region: `US` (default, " + kentikapi.EndpointUS + ") or `EU` (" + kentikapi.EndpointEU + "). " +
				"Ignored when `endpoint` or `apiurl` is set. Can also be specified with KTAPI_REGION environment variable.",
		},
		endpointKey: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_ENDPOINT", nil),
			Description: "Kentik gRPC API endpoint as `host:port`, always over TLS (optional; overrides `region` and " +
				"`apiurl`). Can also be specified with KTAPI_ENDPOINT environment variable.",
		},
		apiURLKey: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_URL", nil),
			Description: "Kentik API base URL (optional), e.g. `https://api.kentik.eu`. The gRPC endpoint is derived " +
				"from it as `grpc.<host>:443`; an `http://` URL selects a plaintext connection (local test servers only). " +
				"Prefer `region` or `endpoint`. Can also be specified with KTAPI_URL environment variable.",
		},
		retryKey: {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Configuration for API client retry mechanism",
			Elem: &schema.Resource{
				Schema: retryProperties(),
			},
		},
		logPayloadsKey: {
			Type:        schema.TypeBool,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_LOG_PAYLOADS", false),
			Description: "Log payloads flag enables verbose debug logs of requests and responses (optional). " +
				"Credentials are never logged. Can also be specified with KTAPI_LOG_PAYLOADS environment variable.",
		},
	}
}

// retryProperties groups retry properties.
// Note that, default values of schema.EnvDefaultFunc are not used, because they are not applied when user does not pass
// retry block at all.
func retryProperties() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		maxAttemptsKey: {
			Type:        schema.TypeInt,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_RETRY_MAX_ATTEMPTS", nil),
			Description: "Maximum number of request retry attempts. " +
				"Minimum valid value: 1 (0 fallbacks to default). Default: 100. " +
				"Can also be specified with KTAPI_RETRY_MAX_ATTEMPTS environment variable.",
		},
		minDelayKey: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_RETRY_MIN_DELAY", nil),
			Description: "Minimum delay before request retry. " +
				"Expected Go time duration format, e.g. 1s (see: <https://pkg.go.dev/time#ParseDuration>). " +
				"Default: 1s (1 second). " +
				"Can also be specified with KTAPI_RETRY_MIN_DELAY environment variable.",
		},
		maxDelayKey: {
			Type:        schema.TypeString,
			Optional:    true,
			DefaultFunc: schema.EnvDefaultFunc("KTAPI_RETRY_MAX_DELAY", nil),
			Description: "Maximum delay before request retry. " +
				"Expected Go time duration format, e.g. 1s (see: <https://pkg.go.dev/time#ParseDuration>). " +
				"Default: 5m (5 minutes). " +
				"Can also be specified with KTAPI_RETRY_MAX_DELAY environment variable.",
		},
	}
}

func clientConfig(ctx context.Context, d *schema.ResourceData, userAgent string) (kentikapi.Config, error) {
	retry, err := getRetryConfig(ctx, d)
	if err != nil {
		return kentikapi.Config{}, err
	}
	cfg := kentikapi.Config{
		Email:     d.Get(emailKey).(string),
		Token:     d.Get(tokenKey).(string),
		Region:    kentikapi.Region(strings.ToUpper(d.Get(regionKey).(string))),
		UserAgent: userAgent,
		Retry:     retry,
	}
	switch endpoint, apiURL := d.Get(endpointKey).(string), d.Get(apiURLKey).(string); {
	case endpoint != "":
		cfg.Endpoint = endpoint
	case apiURL != "":
		cfg.Endpoint, cfg.Insecure, err = kentikapi.EndpointFromAPIURL(apiURL)
		if err != nil {
			return kentikapi.Config{}, err
		}
	}
	if d.Get(logPayloadsKey).(bool) {
		cfg.Log = logPayload
	}
	return cfg, nil
}

func logPayload(ctx context.Context, method string, req, resp any, err error) {
	fields := map[string]any{"method": method, "request": marshalForLog(req)}
	if err != nil {
		fields["error"] = err.Error()
	} else {
		fields["response"] = marshalForLog(resp)
	}
	tflog.Debug(ctx, "Kentik API call", fields)
}

func marshalForLog(v any) string {
	if m, ok := v.(proto.Message); ok {
		if b, err := protojson.Marshal(m); err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}

func getRetryConfig(ctx context.Context, d *schema.ResourceData) (kentikapi.RetryConfig, error) {
	tflog.Debug(ctx, fmt.Sprintf("Getting retry config: %v", d.Get(retryKey)))
	retryCfg, err := getObjectFromNestedResourceData(d.Get(retryKey))
	if err != nil {
		return kentikapi.RetryConfig{}, fmt.Errorf("get retry configuration: %w", err)
	}

	// SDKv2 stores TypeInt as int. (v0.4 asserted uint here, so max_attempts was silently ignored.)
	maxAttempts, ok := retryCfg[maxAttemptsKey].(int)
	if !ok || maxAttempts <= 0 {
		maxAttempts = defaultMaxAttempts
	}

	minDelay, err := parseDelay(retryCfg, minDelayKey, defaultMinDelay)
	if err != nil {
		return kentikapi.RetryConfig{}, err
	}
	maxDelay, err := parseDelay(retryCfg, maxDelayKey, defaultMaxDelay)
	if err != nil {
		return kentikapi.RetryConfig{}, err
	}

	return kentikapi.RetryConfig{
		MaxAttempts: uint(maxAttempts),
		MinDelay:    minDelay,
		MaxDelay:    maxDelay,
	}, nil
}

func parseDelay(cfg map[string]any, key, fallback string) (time.Duration, error) {
	raw, ok := cfg[key].(string)
	if !ok || raw == "" {
		raw = fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %v duration: %w", key, err)
	}
	return v, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func clientFrom(m any) *kentikapi.Client {
	return m.(*kentikapi.Client)
}
