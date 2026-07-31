package datafeed

import "time"

const (
	retryBaseDelay       = time.Second
	retryDelayCap        = 60 * time.Second
	operatorAlertAttempt = 8
)

// RetryDelay implements the locked bounded exponential full-jitter policy.
// randInt receives the exclusive upper bound in nanoseconds so callers can
// use crypto/rand in production and deterministic values in tests.
func RetryDelay(attempt int, randInt func(int64) int64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	limit := retryBaseDelay
	for index := 1; index < attempt && limit < retryDelayCap; index++ {
		limit *= 2
		if limit > retryDelayCap {
			limit = retryDelayCap
		}
	}
	if randInt == nil {
		return 0
	}
	value := randInt(int64(limit))
	if value < 0 || value >= int64(limit) {
		return 0
	}
	return time.Duration(value)
}

func OperatorAlertRequired(attemptCount int) bool {
	return attemptCount >= operatorAlertAttempt
}
