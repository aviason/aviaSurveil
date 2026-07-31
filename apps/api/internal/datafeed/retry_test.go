package datafeed

import (
	"testing"
	"time"
)

func TestRetryDelayUsesLockedFullJitterBoundsAndOperatorAlertThreshold(t *testing.T) {
	if got := RetryDelay(1, func(int64) int64 { return 0 }); got != 0 {
		t.Fatalf("first retry lower bound = %s, want zero", got)
	}
	if got := RetryDelay(7, func(bound int64) int64 { return bound - 1 }); got != 60*time.Second-time.Nanosecond {
		t.Fatalf("seventh retry upper bound = %s, want cap minus one nanosecond", got)
	}
	if got := RetryDelay(20, func(bound int64) int64 { return bound - 1 }); got != 60*time.Second-time.Nanosecond {
		t.Fatalf("capped retry upper bound = %s, want cap minus one nanosecond", got)
	}
	if OperatorAlertRequired(7) {
		t.Fatal("alert triggered before the eighth attempt")
	}
	if !OperatorAlertRequired(8) {
		t.Fatal("alert was not triggered at the eighth attempt")
	}
}
