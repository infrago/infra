package infra

import (
	"os"
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

func TestParseConfigArgsSingleDriverFlag(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	os.Args = []string{"app", "--driver=redis"}
	params := parseConfigArgs()

	if params["driver"] != "redis" {
		t.Fatalf("driver=%v, want redis", params["driver"])
	}
	if file, ok := params["file"]; ok {
		t.Fatalf("file=%v, want no file param", file)
	}
}

func TestParseConfigArgsNormalizesAliases(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	os.Args = []string{"app", "--config-file", "app.yaml", "--redis-addr=redis:6379"}
	params := parseConfigArgs()

	if params["file"] != "app.yaml" {
		t.Fatalf("file=%v, want app.yaml", params["file"])
	}
	if params["addr"] != "redis:6379" {
		t.Fatalf("addr=%v, want redis:6379", params["addr"])
	}
}

func TestDecodeEmptyConfig(t *testing.T) {
	cfg, err := decodeConfig([]byte(" \n\t"), "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || len(cfg) != 0 {
		t.Fatalf("cfg=%v, want empty map", cfg)
	}
	if got := detectConfigFormat([]byte(" \n\t")); got != "" {
		t.Fatalf("format=%q, want empty", got)
	}
}
