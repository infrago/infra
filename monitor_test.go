package infra

import (
	"testing"

	base "github.com/infrago/base"
)

func TestModuleMonitoringEnvelope(t *testing.T) {
	health := NewModuleHealth("cache", true, nil, base.Map{"connections": 2})
	if health.Module != "cache" || health.Status != HealthHealthy || !health.Ready {
		t.Fatalf("unexpected health: %#v", health)
	}
	if health.Details["connections"] != 2 {
		t.Fatalf("unexpected details: %#v", health.Details)
	}

	stats := NewModuleStats("cache", true, base.Map{"hits": int64(3)})
	if stats.Module != "cache" || stats.Status != HealthHealthy || !stats.Ready {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if stats.Metrics["hits"] != int64(3) {
		t.Fatalf("unexpected metrics: %#v", stats.Metrics)
	}
}

func TestUnhealthyModuleIsNeverReady(t *testing.T) {
	health := NewModuleHealth("queue", true, assertError("disconnected"))
	if health.Ready || health.Status != HealthUnhealthy || health.Error == "" {
		t.Fatalf("unexpected health: %#v", health)
	}
}

func TestRuntimeMonitoringIsSortedAndComparable(t *testing.T) {
	runtime := &infragoRuntime{
		modules: []Module{
			&monitorTestModule{name: "zeta", ready: true},
			&monitorTestModule{name: "alpha", ready: true},
		},
		startStatus: true,
	}

	health := runtime.Health()
	if len(health) != 2 || health[0].Module != "alpha" || health[1].Module != "zeta" {
		t.Fatalf("expected sorted health, got %#v", health)
	}
	stats := runtime.Stats()
	if len(stats) != 2 || stats[0].Module != "alpha" || stats[1].Module != "zeta" {
		t.Fatalf("expected sorted stats, got %#v", stats)
	}
	if !runtime.Ready() {
		t.Fatal("expected runtime to be ready")
	}

	runtime.modules[0].(*monitorTestModule).ready = false
	if runtime.Ready() {
		t.Fatal("expected one unready module to make runtime unready")
	}
}

func TestLegacyModuleUsesLifecycleMonitoringFallback(t *testing.T) {
	runtime := &infragoRuntime{
		modules:     []Module{&legacyTestModule{}},
		startStatus: true,
	}

	health := runtime.Health()
	if len(health) != 1 || !health[0].Ready {
		t.Fatalf("expected healthy lifecycle fallback, got %#v", health)
	}
	if health[0].Details["monitor"] != "lifecycle-fallback" {
		t.Fatalf("expected fallback marker, got %#v", health[0].Details)
	}
	if !runtime.Ready() {
		t.Fatal("legacy module must not make a started runtime unready")
	}
	stats := runtime.Stats()
	if len(stats) != 1 || stats[0].Metrics["monitor"] != "lifecycle-fallback" {
		t.Fatalf("expected fallback stats, got %#v", stats)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }

type monitorTestModule struct {
	name  string
	ready bool
}

func (m *monitorTestModule) Register(string, base.Any) {}
func (m *monitorTestModule) Config(base.Map)           {}
func (m *monitorTestModule) Setup()                    {}
func (m *monitorTestModule) Open()                     {}
func (m *monitorTestModule) Start()                    {}
func (m *monitorTestModule) Stop()                     {}
func (m *monitorTestModule) Close()                    {}
func (m *monitorTestModule) Ready() bool               { return m.ready }
func (m *monitorTestModule) Health() ModuleHealth {
	return NewModuleHealth(m.name, m.ready, nil)
}
func (m *monitorTestModule) Stats() ModuleStats {
	return NewModuleStats(m.name, m.ready, base.Map{"requests": int64(1)})
}

type legacyTestModule struct{}

func (*legacyTestModule) Register(string, base.Any) {}
func (*legacyTestModule) Config(base.Map)           {}
func (*legacyTestModule) Setup()                    {}
func (*legacyTestModule) Open()                     {}
func (*legacyTestModule) Start()                    {}
func (*legacyTestModule) Stop()                     {}
func (*legacyTestModule) Close()                    {}
