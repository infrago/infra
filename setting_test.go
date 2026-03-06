package infra

import (
	"testing"

	. "github.com/infrago/base"
)

func TestRuntimeConfigMergesSettingDeeply(t *testing.T) {
	rt := &infragoRuntime{
		project:     INFRAGO,
		profile:     GLOBAL,
		runProfiles: []string{GLOBAL},
		setting: Map{
			"feature": Map{
				"enabled": true,
				"limits": Map{
					"qps": 10,
				},
			},
		},
	}

	rt.runtimeConfig(Map{
		"setting": Map{
			"feature": Map{
				"limits": Map{
					"burst": 20,
				},
			},
			"name": "demo",
		},
	})

	feature, ok := rt.setting["feature"].(Map)
	if !ok {
		t.Fatalf("feature setting missing")
	}
	limits, ok := feature["limits"].(Map)
	if !ok {
		t.Fatalf("limits setting missing")
	}
	if limits["qps"] != 10 {
		t.Fatalf("expected existing nested setting to be preserved, got %v", limits["qps"])
	}
	if limits["burst"] != 20 {
		t.Fatalf("expected nested setting to merge, got %v", limits["burst"])
	}
	if rt.setting["name"] != "demo" {
		t.Fatalf("expected top-level setting to merge, got %v", rt.setting["name"])
	}
}

func TestSettingReturnsDeepCopy(t *testing.T) {
	original := infrago
	infrago = &infragoRuntime{
		project:     INFRAGO,
		profile:     GLOBAL,
		runProfiles: []string{GLOBAL},
		setting: Map{
			"feature": Map{
				"enabled": true,
			},
			"items": []Any{
				Map{"name": "one"},
			},
		},
	}
	defer func() {
		infrago = original
	}()

	setting := Setting()
	feature := setting["feature"].(Map)
	feature["enabled"] = false
	items := setting["items"].([]Any)
	items[0].(Map)["name"] = "changed"

	internalFeature := infrago.setting["feature"].(Map)
	if internalFeature["enabled"] != true {
		t.Fatalf("expected setting copy to be isolated from runtime state")
	}
	internalItems := infrago.setting["items"].([]Any)
	if internalItems[0].(Map)["name"] != "one" {
		t.Fatalf("expected nested slice map copy to be isolated from runtime state")
	}
}
