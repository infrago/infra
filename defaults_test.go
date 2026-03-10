package infra

import (
	"testing"
	"time"
)

func TestDispatchFinal(t *testing.T) {
	retries := []time.Duration{3 * time.Second, 10 * time.Second, 30 * time.Second}
	cases := []struct {
		attempt int
		final   bool
	}{
		{attempt: 1, final: false},
		{attempt: 2, final: false},
		{attempt: 3, final: false},
		{attempt: 4, final: true},
	}
	for _, c := range cases {
		if got := dispatchFinal(retries, c.attempt); got != c.final {
			t.Fatalf("attempt=%d final=%v got=%v", c.attempt, c.final, got)
		}
	}
}

func TestDispatchRetryDelay(t *testing.T) {
	retries := []time.Duration{3 * time.Second, 10 * time.Second, 30 * time.Second}
	cases := []struct {
		attempt int
		delay   time.Duration
		ok      bool
	}{
		{attempt: 1, delay: 3 * time.Second, ok: true},
		{attempt: 2, delay: 10 * time.Second, ok: true},
		{attempt: 3, delay: 30 * time.Second, ok: true},
		{attempt: 4, delay: 0, ok: false},
	}
	for _, c := range cases {
		delay, ok := dispatchRetryDelay(retries, c.attempt)
		if ok != c.ok || delay != c.delay {
			t.Fatalf("attempt=%d delay=%v ok=%v gotDelay=%v gotOK=%v", c.attempt, c.delay, c.ok, delay, ok)
		}
	}
}
