package infra

import (
	. "github.com/infrago/base"
)

// Mount attaches a module into the infrago runtime and returns a host.
func Mount(mod Module) Host {
	return infra.Mount(mod)
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
	infra.setRequestedProfiles(requested)
	infra.Load()
	registry.Apply(infra.EffectiveProfiles()...)
	infra.Setup()
	infra.Open()
}

// Ready is an alias of Prepare for compatibility.
func Ready(profile ...string) {
	Prepare(profile...)
}

// Run starts the full lifecycle and blocks until stop.
func Run(profile ...string) {
	requested := normalizeProfiles(profile...)
	infra.setRequestedProfiles(requested)
	infra.Load()
	registry.Apply(infra.EffectiveProfiles()...)
	infra.Setup()
	infra.Open()
	infra.Start()
	infra.Wait()
	infra.Stop()
	infra.Close()
}

// Go is an alias of Run for compatibility.
func Go(profile ...string) {
	Run(profile...)
}

// Override controls whether registrations can overwrite existing entries.
func Override(args ...bool) bool {
	return infra.Override(args...)
}

func Identity() infragoIdentity {
	return infra.Identity()
}

func Node() string {
	return infra.Node()
}

// Invoke executes one entry as a new request context.
func Invoke(name string, values ...Map) (Map, Res) {
	var value Map
	if len(values) > 0 {
		value = values[0]
	}
	return core.Invoke(nil, name, value)
}

// Invokes executes multiple entries and returns results in order.
func Invokes(name string, values ...Map) ([]Map, Res) {
	if len(values) == 0 {
		return []Map{}, OK
	}
	results := make([]Map, 0, len(values))
	for _, value := range values {
		data, res := Invoke(name, value)
		if res != nil && res.Fail() {
			return results, res
		}
		results = append(results, data)
	}
	return results, OK
}

// Invoking executes a paged subset of calls and returns total input count.
func Invoking(name string, offset, limit int, values ...Map) (int64, []Map) {
	total := int64(len(values))
	start, end := normalizeInvokeWindow(len(values), offset, limit)
	if start >= end {
		return total, []Map{}
	}
	results := make([]Map, 0, end-start)
	for _, value := range values[start:end] {
		data, res := Invoke(name, value)
		if res != nil && res.Fail() {
			return total, results
		}
		results = append(results, data)
	}
	return total, results
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
