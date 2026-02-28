package bamgoo

import . "github.com/bamgoo/base"

const (
	TraceKindInternal = "internal"
	TraceKindServer   = "server"
	TraceKindConsumer = "consumer"
)

// TraceAttrs builds a normalized trace attrs map.
func TraceAttrs(service, kind, target string, attrs ...Map) Map {
	out := Map{
		"service": service,
		"kind":    kind,
	}
	if target != "" {
		out["target"] = target
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
