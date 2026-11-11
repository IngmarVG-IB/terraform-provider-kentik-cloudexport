package kentikapi

import "time"

// Backoff exposes RetryConfig.backoff (with defaults applied) to tests.
func Backoff(r RetryConfig, n uint) time.Duration { return r.withDefaults().backoff(n) }
