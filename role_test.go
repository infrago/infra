package infra

import (
	"testing"

	. "github.com/infrago/base"
)

func TestRuntimeConfigReadsRole(t *testing.T) {
	rt := &infragoRuntime{
		project:     INFRAGO,
		profile:     GLOBAL,
		runProfiles: []string{GLOBAL},
		setting:     Map{},
	}

	rt.runtimeConfig(Map{
		"role": "site",
	})
	if rt.configRole != "site" {
		t.Fatalf("expected config role site, got %q", rt.configRole)
	}

	rt.runtimeConfig(Map{
		"infrago": Map{
			"role": "worker",
		},
	})
	if rt.configRole != "worker" {
		t.Fatalf("expected nested infrago.role to override, got %q", rt.configRole)
	}
}

func TestEffectiveRoleFallsBackToProfile(t *testing.T) {
	rt := &infragoRuntime{
		project:     INFRAGO,
		profile:     GLOBAL,
		runProfiles: []string{"site-sys"},
		setting:     Map{},
	}

	if role := rt.effectiveRoleLocked("site-sys"); role != "site-sys" {
		t.Fatalf("expected fallback role to equal profile, got %q", role)
	}

	rt.configRole = "site"
	if role := rt.effectiveRoleLocked("site-sys"); role != "site" {
		t.Fatalf("expected explicit role to win, got %q", role)
	}
}
