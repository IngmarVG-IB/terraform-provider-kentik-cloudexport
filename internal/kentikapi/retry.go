package kentikapi

import (
	"context"
	"math/rand/v2"
	"slices"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Retry defaults used for zero-valued RetryConfig fields.
const (
	DefaultMaxAttempts = 5
	DefaultMinDelay    = time.Second
	DefaultMaxDelay    = 30 * time.Second
)

// DefaultRetryCodes are the gRPC codes that are retried: the server or a proxy
// was unavailable, or a rate limit was hit. Neither means the request was applied.
var DefaultRetryCodes = []codes.Code{codes.Unavailable, codes.ResourceExhausted} //nolint:gochecknoglobals

// RetryConfig controls retries with exponential backoff and full jitter.
type RetryConfig struct {
	// MaxAttempts is the total number of attempts, including the first one.
	// 1 disables retries. Zero means DefaultMaxAttempts.
	MaxAttempts uint
	// MinDelay is the base delay before the first retry. Zero means DefaultMinDelay.
	MinDelay time.Duration
	// MaxDelay caps the delay between attempts. Zero means DefaultMaxDelay.
	MaxDelay time.Duration
	// Codes are the retryable status codes. Nil means DefaultRetryCodes.
	Codes []codes.Code
}

func (r RetryConfig) withDefaults() RetryConfig {
	if r.MaxAttempts == 0 {
		r.MaxAttempts = DefaultMaxAttempts
	}
	if r.MinDelay <= 0 {
		r.MinDelay = DefaultMinDelay
	}
	if r.MaxDelay <= 0 {
		r.MaxDelay = DefaultMaxDelay
	}
	if r.MaxDelay < r.MinDelay {
		r.MaxDelay = r.MinDelay
	}
	if r.Codes == nil {
		r.Codes = DefaultRetryCodes
	}
	return r
}

// backoff returns the delay before retry number n (1-based): a random value in
// [MinDelay, min(MaxDelay, MinDelay*2^(n-1))].
func (r RetryConfig) backoff(n uint) time.Duration {
	ceiling := r.MinDelay
	for i := uint(1); i < n && ceiling < r.MaxDelay; i++ {
		ceiling *= 2
	}
	ceiling = min(ceiling, r.MaxDelay)
	spread := ceiling - r.MinDelay
	if spread <= 0 {
		return r.MinDelay
	}
	return r.MinDelay + rand.N(spread+1) //nolint:gosec // jitter does not need a CSPRNG
}

func retryInterceptor(r RetryConfig) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		var err error
		for attempt := uint(1); ; attempt++ {
			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil || attempt >= r.MaxAttempts || !slices.Contains(r.Codes, status.Code(err)) {
				return err
			}
			t := time.NewTimer(r.backoff(attempt))
			select {
			case <-ctx.Done():
				t.Stop()
				return err
			case <-t.C:
			}
		}
	}
}
