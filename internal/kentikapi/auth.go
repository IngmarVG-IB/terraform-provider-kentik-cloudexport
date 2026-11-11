package kentikapi

import "context"

// Metadata keys used by the Kentik API (see the security_definitions in the
// api-schema-public protos). gRPC metadata keys are case-insensitive and sent
// lower-cased on the wire.
const (
	MetadataAuthEmail = "X-CH-Auth-Email"
	MetadataAuthToken = "X-CH-Auth-API-Token" //nolint:gosec // header name, not a credential
)

// authCredentials implements credentials.PerRPCCredentials.
type authCredentials struct {
	email  string
	token  string
	secure bool
}

func (a authCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		MetadataAuthEmail: a.email,
		MetadataAuthToken: a.token,
	}, nil
}

// RequireTransportSecurity keeps the token off plaintext connections unless the
// caller explicitly asked for an insecure connection (local fakes only).
func (a authCredentials) RequireTransportSecurity() bool { return a.secure }
