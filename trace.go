package infra

import "strings"

import . "github.com/infrago/base"

const (
	TraceKindMethod  = "method"
	TraceKindService = "service"
	TraceKindTrigger = "trigger"
	TraceKindHTTP    = "http"
	TraceKindWeb     = "web"
	TraceKindEvent   = "event"
	TraceKindQueue   = "queue"
	TraceKindCron    = "cron"
	TraceKindCustom  = "custom"
)

// TraceAttrs builds a normalized trace attrs map.
func TraceAttrs(service, kind, entry string, attrs ...Map) Map {
	k, e := normalizeTraceKindEntry(kind, entry)
	out := Map{
		"service": service,
		"kind":    k,
		"entry":   e,
		"step":    "internal",
	}
	for _, item := range attrs {
		if item == nil {
			continue
		}
		for k, v := range item {
			out[k] = v
		}
	}
	return out
}

func normalizeTraceKindEntry(kind, entry string) (string, string) {
	k := strings.ToLower(strings.TrimSpace(kind))
	e := strings.TrimSpace(entry)
	switch k {
	case TraceKindMethod, TraceKindService, TraceKindTrigger, TraceKindHTTP, TraceKindWeb, TraceKindEvent, TraceKindQueue, TraceKindCron:
		if e == "" {
			e = "unknown"
		}
		return k, e
	default:
		if e == "" {
			e = k
		}
		if e == "" {
			e = "custom"
		}
		return TraceKindCustom, e
	}
}
