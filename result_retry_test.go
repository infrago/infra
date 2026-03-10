package infra

import "testing"

func TestRetryResultFlag(t *testing.T) {
	res := Fail.With("boom").Retry()
	if !IsRetry(res) {
		t.Fatalf("expected retry flag")
	}
	if msg := res.Error(); msg == "" {
		t.Fatalf("expected explicit error message")
	}
}

func TestWithKeepsRetryFlag(t *testing.T) {
	res := Fail.Retry().With("db timeout")
	if !IsRetry(res) {
		t.Fatalf("expected retry flag after With")
	}
}

func TestRetryResultHelper(t *testing.T) {
	res := RetryResult(Fail, "network broken")
	if !IsRetry(res) {
		t.Fatalf("expected retry flag from helper")
	}
}
