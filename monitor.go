package infra

import (
	"reflect"
	"sort"
	"strings"
	"time"

	base "github.com/infrago/base"
)

const (
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
)

// ModuleHealth is the common health/readiness representation for every module.
type ModuleHealth struct {
	Module    string    `json:"module"`
	Status    string    `json:"status"`
	Ready     bool      `json:"ready"`
	Project   string    `json:"project"`
	Role      string    `json:"role"`
	Profile   string    `json:"profile"`
	Node      string    `json:"node"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
	Details   base.Map  `json:"details,omitempty"`
}

// ModuleStats is the common statistics envelope. Module-specific metrics stay
// under Metrics so callers can compare identity, health and readiness directly.
type ModuleStats struct {
	Module      string    `json:"module"`
	Status      string    `json:"status"`
	Ready       bool      `json:"ready"`
	Project     string    `json:"project"`
	Role        string    `json:"role"`
	Profile     string    `json:"profile"`
	Node        string    `json:"node"`
	CollectedAt time.Time `json:"collected_at"`
	Metrics     base.Map  `json:"metrics"`
}

// ModuleMonitor is implemented by modules that expose detailed observability.
// It remains optional so independently versioned Infrago drivers can be upgraded
// without breaking the core lifecycle contract.
type ModuleMonitor interface {
	Health() ModuleHealth
	Ready() bool
	Stats() ModuleStats
}

func moduleMonitorName(module base.Any) string {
	typ := reflect.TypeOf(module)
	if typ == nil {
		return "unknown"
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if path := typ.PkgPath(); path != "" {
		parts := strings.Split(path, "/")
		return normalizeLogModule(parts[len(parts)-1])
	}
	if name := typ.Name(); name != "" {
		return normalizeLogModule(name)
	}
	return "unknown"
}

// NewModuleHealth builds a normalized health result for a module.
func NewModuleHealth(module string, ready bool, err error, details ...base.Map) ModuleHealth {
	identity := Identity()
	status := HealthHealthy
	if !ready || err != nil {
		status = HealthUnhealthy
		ready = false
	}
	health := ModuleHealth{
		Module:    normalizeLogModule(module),
		Status:    status,
		Ready:     ready,
		Project:   identity.Project,
		Role:      identity.Role,
		Profile:   identity.Profile,
		Node:      identity.Node,
		CheckedAt: time.Now(),
		Details:   mergeMonitorMaps(details...),
	}
	if err != nil {
		health.Error = err.Error()
	}
	return health
}

// NewModuleStats builds a normalized statistics result for a module.
func NewModuleStats(module string, ready bool, metrics ...base.Map) ModuleStats {
	identity := Identity()
	status := HealthUnhealthy
	if ready {
		status = HealthHealthy
	}
	return ModuleStats{
		Module:      normalizeLogModule(module),
		Status:      status,
		Ready:       ready,
		Project:     identity.Project,
		Role:        identity.Role,
		Profile:     identity.Profile,
		Node:        identity.Node,
		CollectedAt: time.Now(),
		Metrics:     mergeMonitorMaps(metrics...),
	}
}

func mergeMonitorMaps(items ...base.Map) base.Map {
	out := base.Map{}
	for _, item := range items {
		for key, value := range item {
			out[key] = value
		}
	}
	return out
}

func normalizeModuleHealth(health ModuleHealth) ModuleHealth {
	health.Module = normalizeLogModule(health.Module)
	if health.CheckedAt.IsZero() {
		health.CheckedAt = time.Now()
	}
	identity := Identity()
	if strings.TrimSpace(health.Project) == "" {
		health.Project = identity.Project
	}
	if strings.TrimSpace(health.Role) == "" {
		health.Role = identity.Role
	}
	if strings.TrimSpace(health.Profile) == "" {
		health.Profile = identity.Profile
	}
	if strings.TrimSpace(health.Node) == "" {
		health.Node = identity.Node
	}
	if health.Status == "" {
		if health.Ready && health.Error == "" {
			health.Status = HealthHealthy
		} else {
			health.Status = HealthUnhealthy
		}
	}
	if health.Details == nil {
		health.Details = base.Map{}
	}
	return health
}

func normalizeModuleStats(stats ModuleStats) ModuleStats {
	stats.Module = normalizeLogModule(stats.Module)
	if stats.CollectedAt.IsZero() {
		stats.CollectedAt = time.Now()
	}
	identity := Identity()
	if strings.TrimSpace(stats.Project) == "" {
		stats.Project = identity.Project
	}
	if strings.TrimSpace(stats.Role) == "" {
		stats.Role = identity.Role
	}
	if strings.TrimSpace(stats.Profile) == "" {
		stats.Profile = identity.Profile
	}
	if strings.TrimSpace(stats.Node) == "" {
		stats.Node = identity.Node
	}
	if stats.Status == "" {
		if stats.Ready {
			stats.Status = HealthHealthy
		} else {
			stats.Status = HealthUnhealthy
		}
	}
	if stats.Metrics == nil {
		stats.Metrics = base.Map{}
	}
	return stats
}

func sortModuleHealth(items []ModuleHealth) {
	sort.Slice(items, func(i, j int) bool { return items[i].Module < items[j].Module })
}

func sortModuleStats(items []ModuleStats) {
	sort.Slice(items, func(i, j int) bool { return items[i].Module < items[j].Module })
}
