package infra

import "testing"

func TestScopedName(t *testing.T) {
	if got := scopedName("www", "index"); got != "www.index" {
		t.Fatalf("expected www.index, got %s", got)
	}
	if got := scopedName("www", "www.login"); got != "www.login" {
		t.Fatalf("expected unchanged full key, got %s", got)
	}
	if got := scopedName("www", ".index"); got != ".index" {
		t.Fatalf("expected unchanged .index, got %s", got)
	}
	if got := scopedName("www", "*.access"); got != "*.access" {
		t.Fatalf("expected unchanged *.access, got %s", got)
	}
	if got := scopedName("www", ""); got != "www" {
		t.Fatalf("expected scope key www for empty name, got %s", got)
	}
}
