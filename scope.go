package infra

import (
	"strings"

	. "github.com/infrago/base"
)

// Scope creates a named registration scope so names can omit repeated prefix.
func Scope(name string) *registerScope {
	return &registerScope{name: normalizeScopeName(name)}
}

type registerScope struct {
	name string
}

// Register behaves like infra.Register within the scope prefix.
// Example: Scope("www").Register("index", web.Router{}) => register "www.index".
func (s *registerScope) Register(args ...Any) {
	name := ""
	values := make([]Any, 0)
	for _, arg := range args {
		if v, ok := arg.(string); ok {
			name = v
		} else {
			values = append(values, arg)
		}
	}

	target := scopedName(s.name, name)
	for _, value := range values {
		Register(target, value)
	}
}

func normalizeScopeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func scopedName(scope, name string) string {
	scope = normalizeScopeName(scope)
	name = strings.TrimSpace(strings.ToLower(name))

	if scope == "" {
		return name
	}
	if name == "" {
		return scope
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "*.") {
		return name
	}
	if strings.HasPrefix(name, scope+".") {
		return name
	}
	return scope + "." + name
}
