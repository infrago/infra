package infra

import "testing"

func TestMetaResultChecksDoNotConsumePendingResult(t *testing.T) {
	meta := NewMeta()
	meta.Result(Denied)

	if meta.ResultOK() {
		t.Fatal("expected denied result not to be OK")
	}
	if !meta.ResultFail() {
		t.Fatal("expected denied result to fail")
	}
	if got := meta.Result(); got != Denied {
		t.Fatalf("expected pending denied result after checks, got %#v", got)
	}
	if !meta.ResultOK() || meta.ResultFail() {
		t.Fatal("expected consumed result to return to implicit OK")
	}
}
