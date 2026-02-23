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
		bamgoo.Register(name, value)
	}
}

// Prepare initializes and opens modules without starting them.
func Prepare(role ...string) {
	if len(role) > 0 {
		bamgoo.setRole(role[0])
	}
	bamgoo.Load()
	bamgoo.Setup()
	bamgoo.Open()
}

// Ready is an alias of Prepare for compatibility.
func Ready(role ...string) {
	Prepare(role...)
}

// Run starts the full lifecycle and blocks until stop.
func Run(role ...string) {
	if len(role) > 0 {
		bamgoo.setRole(role[0])
	}
	bamgoo.Load()
	bamgoo.Setup()
	bamgoo.Open()
	bamgoo.Start()
	bamgoo.Wait()
	bamgoo.Stop()
	bamgoo.Close()
}

// Go is an alias of Run for compatibility.
func Go(role ...string) {
	Run(role...)
}

// Override controls whether registrations can overwrite existing entries.
func Override(args ...bool) bool {
	return bamgoo.Override(args...)
}

func Identity() bamgooIdentity {
	return bamgoo.Identity()
}

func Name() string {
	return bamgoo.Name()
}

func Project() string {
	return bamgoo.Project()
}

func Role() string {
	return bamgoo.Role()
}

func Node() string {
	return bamgoo.Node()
}

func Version() string {
	return bamgoo.Version()
}
