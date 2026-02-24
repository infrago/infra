package bamgoo

import (
	. "github.com/bamgoo/base"
)

// Mount attaches a module into the bamgoo runtime and returns a host.
func Mount(mod Module) Host {
	return bamgoo.Mount(mod)
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
	selected := normalizeProfiles(profile...)
	bamgoo.setProfile(selected[0])
	registry.Apply(selected...)
	bamgoo.Load()
	bamgoo.Setup()
	bamgoo.Open()
}

// Ready is an alias of Prepare for compatibility.
func Ready(profile ...string) {
	Prepare(profile...)
}

// Run starts the full lifecycle and blocks until stop.
func Run(profile ...string) {
	selected := normalizeProfiles(profile...)
	bamgoo.setProfile(selected[0])
	registry.Apply(selected...)
	bamgoo.Load()
	bamgoo.Setup()
	bamgoo.Open()
	bamgoo.Start()
	bamgoo.Wait()
	bamgoo.Stop()
	bamgoo.Close()
}

// Go is an alias of Run for compatibility.
func Go(profile ...string) {
	Run(profile...)
}

// Override controls whether registrations can overwrite existing entries.
func Override(args ...bool) bool {
	return bamgoo.Override(args...)
}

func Identity() bamgooIdentity {
	return bamgoo.Identity()
}

func Node() string {
	return bamgoo.Node()
}

// Invoke executes one entry as a new request context.
func Invoke(name string, value Map) (Map, Res) {
	return core.Invoke(nil, name, value)
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
