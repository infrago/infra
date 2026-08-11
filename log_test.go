package infra

import (
	"bytes"
	"encoding/json"
	"testing"

	base "github.com/infrago/base"
)

type captureLogHook struct {
	entries []LogEntry
}

func (h *captureLogHook) Log(entry LogEntry) {
	h.entries = append(h.entries, entry)
}

type unavailableLogHook struct {
	captureLogHook
}

func (h *unavailableLogHook) Ready() bool { return false }

func TestStructuredLogIncludesIdentityAndMetadata(t *testing.T) {
	hook.mutex.Lock()
	original := hook.log
	capture := &captureLogHook{}
	hook.log = capture
	hook.mutex.Unlock()
	t.Cleanup(func() {
		hook.mutex.Lock()
		hook.log = original
		hook.mutex.Unlock()
	})

	meta := NewMeta()
	meta.RequestId("request-1")
	meta.TraceId("0123456789abcdef0123456789abcdef")
	LogWith(meta, LogLevelWarning, "queue", "delayed", base.Map{"attempt": 2})

	if len(capture.entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(capture.entries))
	}
	entry := capture.entries[0]
	if entry.Module != "queue" || entry.RequestID != "request-1" {
		t.Fatalf("unexpected log identity: %#v", entry)
	}
	if entry.TraceID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected trace id: %q", entry.TraceID)
	}
	if entry.Node != Identity().Node {
		t.Fatalf("expected runtime node %q, got %q", Identity().Node, entry.Node)
	}
	if entry.Fields["attempt"] != 2 {
		t.Fatalf("unexpected fields: %#v", entry.Fields)
	}
}

func TestDefaultLogHookWritesJSON(t *testing.T) {
	var stdout bytes.Buffer
	logger := &defaultLogHook{stdout: &stdout, stderr: &stdout}
	logger.Log(LogEntry{Level: LogLevelInfo, Module: "infra", Message: "ready"})

	entry := LogEntry{}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &entry); err != nil {
		t.Fatalf("expected JSON log: %v", err)
	}
	if entry.Module != "infra" || entry.Message != "ready" {
		t.Fatalf("unexpected JSON log: %#v", entry)
	}
}

func TestRequestIDPropagatesThroughMetadata(t *testing.T) {
	source := NewMeta()
	source.RequestId("request-2")

	target := NewMeta()
	target.Metadata(source.Metadata())
	if got := target.RequestId(); got != "request-2" {
		t.Fatalf("expected propagated request id, got %q", got)
	}
}

func TestUnavailableLogHookUsesFallback(t *testing.T) {
	primary := &unavailableLogHook{}
	fallback := &captureLogHook{}
	logger := &infragoHook{log: primary, logFallback: fallback}

	logger.Log(LogEntry{Level: LogLevelInfo, Module: "infra", Message: "stopped"})

	if len(primary.entries) != 0 {
		t.Fatalf("expected unavailable primary logger to be skipped, got %d entries", len(primary.entries))
	}
	if len(fallback.entries) != 1 || fallback.entries[0].Message != "stopped" {
		t.Fatalf("expected fallback log entry, got %#v", fallback.entries)
	}
}
