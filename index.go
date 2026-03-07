package infra

import (
	. "github.com/infrago/base"
)

// Mount attaches a module into the infrago runtime and returns a host.
func Mount(mod Module) Host {
	return infrago.Mount(mod)
}

// Register registers anything into mounted modules.
func Register(args ...Any) {
	name := ""
	values := make([]Any, 0)
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			name = v
		default:
			values = append(values, v)
		}
	}

	for _, value := range values {
		registry.Register(name, value)
	}
}

func RegisterProfile(key string, profile Profile) {
	registry.RegisterProfile(key, profile)
}

// Prepare initializes and opens modules without starting them.
func Prepare(profile ...string) {
	requested := normalizeProfiles(profile...)
	infrago.setRequestedProfiles(requested)
	infrago.Load()
	registry.Apply(infrago.EffectiveProfiles()...)
	infrago.Setup()
	infrago.Open()
}

// Ready is an alias of Prepare for compatibility.
func Ready(profile ...string) {
	Prepare(profile...)
}

// Run starts the full lifecycle and blocks until stop.
func Run(profile ...string) {
	requested := normalizeProfiles(profile...)
	infrago.setRequestedProfiles(requested)
	infrago.Load()
	registry.Apply(infrago.EffectiveProfiles()...)
	infrago.Setup()
	infrago.Open()
	infrago.Start()
	infrago.Wait()
	infrago.Stop()
	infrago.Close()
}

// Go is an alias of Run for compatibility.
func Go(profile ...string) {
	Run(profile...)
}

// Override controls whether registrations can overwrite existing entries.
func Override(args ...bool) bool {
	return infrago.Override(args...)
}

func Setting() Map {
	return infrago.Setting()
}

func Identity() infragoIdentity {
	return infrago.Identity()
}

func Node() string {
	return infrago.Node()
}

func Arguments(name string, extends ...Vars) Vars {
	return core.Arguments(name, extends...)
}

// Invoke executes one entry as a new request context.
func Invoke(name string, values ...Map) (Map, Res) {
	var value Map
	if len(values) > 0 {
		value = values[0]
	}
	return core.Invoke(nil, name, value)
}

// InvokeList executes one entry and returns response data with parsed "items" list.
func InvokeList(name string, values ...Map) (Map, []Map) {
	data, _ := Invoke(name, values...)
	return invokeListData(data)
}

// Invokes executes one entry and returns response items list.
func Invokes(name string, values ...Map) ([]Map, Res) {
	data, res := Invoke(name, values...)
	return invokeItems(data), res
}

// Invoking executes one entry and returns paged items with total count.
func Invoking(name string, offset, limit int, values ...Map) (int64, []Map) {
	data, _ := Invoke(name, values...)
	items := invokeItems(data)
	if total, ok := invokeTotal(data); ok {
		return total, items
	}
	total := int64(len(items))
	start, end := normalizeInvokeWindow(len(items), offset, limit)
	if start >= end {
		return total, []Map{}
	}
	return total, items[start:end]
}

// InvokeOK executes one entry and returns whether result is OK.
func InvokeOK(name string, values ...Map) bool {
	_, res := Invoke(name, values...)
	return res == nil || res.OK()
}

// InvokeFail executes one entry and returns whether result is failed.
func InvokeFail(name string, values ...Map) bool {
	return !InvokeOK(name, values...)
}

func normalizeInvokeWindow(total, offset, limit int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return total, total
	}
	end := total
	if limit > 0 {
		end = offset + limit
		if end > total {
			end = total
		}
	}
	return offset, end
}

func invokeItems(data Map) []Map {
	if data == nil {
		return []Map{}
	}
	raw, ok := data["items"]
	if !ok || raw == nil {
		return []Map{}
	}
	switch items := raw.(type) {
	case []Map:
		return items
	case []Any:
		out := make([]Map, 0, len(items))
		for _, item := range items {
			switch v := item.(type) {
			case Map:
				out = append(out, v)
			}
		}
		return out
	default:
		return []Map{}
	}
}

func invokeItem(data Map) Map {
	if data == nil {
		return Map{}
	}
	if raw, ok := data["item"]; ok {
		if item, ok := raw.(Map); ok && item != nil {
			return item
		}
		return Map{}
	}
	item := Map{}
	for k, v := range data {
		if k == "items" {
			continue
		}
		item[k] = v
	}
	return item
}

func invokeListData(data Map) (Map, []Map) {
	return invokeItem(data), invokeItems(data)
}

func invokeTotal(data Map) (int64, bool) {
	if data == nil {
		return 0, false
	}
	raw, ok := data["total"]
	if !ok || raw == nil {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// Enqueue dispatches one async queued service request.
func Enqueue(name string, value Map) error {
	return hook.Enqueue(name, value)
}

// Broadcast dispatches one async event to all subscribers.
func Broadcast(name string, value Map) error {
	return hook.Broadcast(name, value)
}

// Publish dispatches one async event (currently same behavior as Broadcast).
func Publish(name string, value Map) error {
	return hook.Publish(name, value)
}

func normalizeProfiles(profile ...string) []string {
	if len(profile) == 0 {
		return []string{GLOBAL}
	}
	out := make([]string, 0, len(profile))
	for _, name := range profile {
		name = normalizeToken(name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		return []string{GLOBAL}
	}
	return out
}
