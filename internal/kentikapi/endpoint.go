package kentikapi

import (
	"fmt"
	"net"
	"net/url"
)

// EndpointFromAPIURL derives a gRPC target from a legacy Kentik API base URL
// (the `apiurl` / KTAPI_URL setting of the older providers and community SDKs),
// keeping the exact rules of github.com/kentik/community_sdk_golang:
//
//   - https://api.kentik.eu      -> grpc.api.kentik.eu:443, TLS
//   - https://host:8443          -> grpc.host:8443, TLS
//   - http://127.0.0.1:9955      -> 127.0.0.1:9955, plaintext (IP literals get no "grpc." prefix)
//   - http://host                -> grpc.host:80, plaintext
func EndpointFromAPIURL(apiURL string) (endpoint string, insecureTransport bool, err error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return "", false, fmt.Errorf("parse API URL %q: %w", apiURL, err)
	}
	var defaultPort string
	switch u.Scheme {
	case "https":
		defaultPort = "443"
	case "http":
		defaultPort = "80"
		insecureTransport = true
	default:
		return "", false, fmt.Errorf("API URL %q: scheme must be http or https", apiURL)
	}
	host := u.Hostname()
	if host == "" {
		return "", false, fmt.Errorf("API URL %q has no host", apiURL)
	}
	port := u.Port()
	if port == "" {
		port = defaultPort
	}
	if net.ParseIP(host) == nil {
		host = "grpc." + host
	}
	return net.JoinHostPort(host, port), insecureTransport, nil
}
