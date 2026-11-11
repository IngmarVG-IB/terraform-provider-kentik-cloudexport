package provider_test

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/kentik/terraform-provider-kentik-cloudexport/internal/provider"
)

func TestProvider_InternalValidate(t *testing.T) {
	t.Parallel()
	require.NoError(t, provider.New().InternalValidate())
}

func TestProvider_Configure(t *testing.T) {
	t.Parallel()
	const trigger = `
data "kentik-cloudexport_item" "aws" {
  id = "1"
}
`
	tests := map[string]struct {
		config  string
		wantErr string
	}{
		"minimal": {config: providerConfig},
		"custom retry and EU region": {config: `
provider "kentik-cloudexport" {
  email        = "joe.doe@example.com"
  token        = "dummy-token"
  region       = "eu"
  log_payloads = true
  retry {
    max_attempts = 66
    min_delay    = "100ms"
    max_delay    = "1m"
  }
}
`},
		"legacy apiurl": {config: `
provider "kentik-cloudexport" {
  email  = "joe.doe@example.com"
  token  = "dummy-token"
  apiurl = "https://api.kentik.eu"
}
`},
		"invalid retry delay": {
			config: `
provider "kentik-cloudexport" {
  email = "joe.doe@example.com"
  token = "dummy-token"
  retry {
    max_delay = "invalid-delay"
  }
}
`,
			wantErr: "parse max_delay duration",
		},
		"invalid region": {
			config: `
provider "kentik-cloudexport" {
  email  = "joe.doe@example.com"
  token  = "dummy-token"
  region = "APAC"
}
`,
			wantErr: `expected region to be one of`,
		},
		"invalid apiurl": {
			config: `
provider "kentik-cloudexport" {
  email  = "joe.doe@example.com"
  token  = "dummy-token"
  apiurl = "api.kentik.com"
}
`,
			wantErr: "scheme must be http or https",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv, _ := newFake(t, seedExports()...)
			step := resource.TestStep{Config: tc.config + trigger}
			if tc.wantErr != "" {
				step.ExpectError = regexp.MustCompile(tc.wantErr)
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: factories(srv),
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}

// TestProvider_SendsConfiguredCredentials checks the whole path from HCL
// provider settings to gRPC metadata on the wire.
func TestProvider_SendsConfiguredCredentials(t *testing.T) {
	t.Parallel()
	srv, _ := newFake(t, seedExports()...)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories(srv),
		Steps: []resource.TestStep{{
			Config: `
provider "kentik-cloudexport" {
  email = "alice@example.com"
  token = "secret-token-123"
}

data "kentik-cloudexport_list" "all" {}
`,
			Check: func(*terraform.State) error {
				calls := srv.Calls()
				if len(calls) == 0 {
					return errors.New("no RPCs received")
				}
				for _, c := range calls {
					if got := c.Metadata.Get("x-ch-auth-email"); len(got) != 1 || got[0] != "alice@example.com" {
						return fmt.Errorf("%s: x-ch-auth-email = %v", c.Method, got)
					}
					if got := c.Metadata.Get("x-ch-auth-api-token"); len(got) != 1 || got[0] != "secret-token-123" {
						return fmt.Errorf("%s: x-ch-auth-api-token = %v", c.Method, got)
					}
					ua := strings.Join(c.Metadata.Get("user-agent"), " ")
					if !strings.Contains(ua, "terraform-provider-kentik-cloudexport/") || !strings.Contains(ua, "Terraform/") {
						return fmt.Errorf("%s: user-agent = %q", c.Method, ua)
					}
				}
				return nil
			},
		}},
	})
}
