package bamgoo

import (
	"path"
	"sort"
	"strings"
	"sync"

	. "github.com/bamgoo/base"
)

var registry = &registerRegistry{
	entries: make([]registerEntry, 0),
	profiles: map[string]Profile{
		GLOBAL: {
			Name:     "全局",
			Desc:     "默认配置",
			Includes: []string{"*"},
		},
	},
}

type (
	Profile struct {
		Name     string
		Desc     string
		Includes []string
		Exclues  []string
	}

	RegistryComponent interface {
		RegistryComponent() string
	}

	registerEntry struct {
		key   string
		name  string
		value Any
	}

	registerRegistry struct {
		mutex sync.RWMutex

		entries  []registerEntry
		profiles map[string]Profile
		applied  bool
	}
)

func (r *registerRegistry) Register(name string, value Any) {
	if value == nil {
		return
	}

	if profile, ok := value.(Profile); ok {
		r.RegisterProfile(name, profile)
		return
	}

	component := ""
	if cc, ok := value.(RegistryComponent); ok {
		component = normalizeToken(cc.RegistryComponent())
	}

	// no component => keep existing behavior (register immediately)
	if component == "" {
		bamgoo.Register(name, value)
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.applied {
		bamgoo.Register(name, value)
		return
	}

	r.entries = append(r.entries, registerEntry{
		key:   buildRegistryKey(component, name),
		name:  name,
		value: value,
	})
}

func (r *registerRegistry) RegisterProfile(key string, profile Profile) {
	key = normalizeToken(key)
	if key == "" {
		return
	}
	if strings.TrimSpace(profile.Name) == "" {
		profile.Name = key
	}
	profile.Includes = normalizePatterns(profile.Includes)
	profile.Exclues = normalizePatterns(profile.Exclues)

	r.mutex.Lock()
	r.profiles[key] = profile
	r.mutex.Unlock()
}

func (r *registerRegistry) Apply(selected ...string) {
	r.mutex.Lock()
	if r.applied {
		r.mutex.Unlock()
		return
	}

	entries := make([]registerEntry, len(r.entries))
	copy(entries, r.entries)
	profiles := make(map[string]Profile, len(r.profiles))
	for k, v := range r.profiles {
		profiles[k] = v
	}
	r.applied = true
	r.mutex.Unlock()

	if len(selected) == 0 || (len(selected) == 1 && strings.TrimSpace(selected[0]) == "") {
		selected = []string{GLOBAL}
	}

	matchers := buildMatchers(selected, profiles)
	for _, entry := range entries {
		if len(matchers) > 0 && !matchesAny(entry.key, matchers) {
			continue
		}
		bamgoo.Register(entry.name, entry.value)
	}
}

type profileMatcher struct {
	include []string
	exclude []string
}

func buildMatchers(selected []string, profiles map[string]Profile) []profileMatcher {
	if len(selected) == 0 {
		return nil // empty => include all
	}

	matchers := make([]profileMatcher, 0, len(selected))
	for _, raw := range selected {
		name := normalizeToken(raw)
		if name == "" {
			continue
		}

		if profile, ok := profiles[name]; ok {
			include := profile.Includes
			if len(include) == 0 {
				include = []string{"*"}
			}
			matchers = append(matchers, profileMatcher{
				include: include,
				exclude: profile.Exclues,
			})
			continue
		}

		// allow direct pattern in Run/Prepare args.
		matchers = append(matchers, profileMatcher{
			include: []string{name},
		})
	}

	return matchers
}

func matchesAny(name string, matchers []profileMatcher) bool {
	for _, matcher := range matchers {
		if !matchesPatternList(name, matcher.include) {
			continue
		}
		if matchesPatternList(name, matcher.exclude) {
			continue
		}
		return true
	}
	return false
}

func matchesPatternList(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, raw := range patterns {
		pattern := normalizeToken(raw)
		if pattern == "" {
			continue
		}
		if pattern == "*" || pattern == name {
			return true
		}
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, raw := range patterns {
		p := normalizeToken(raw)
		if p == "" {
			continue
		}
		if _, ok := uniq[p]; ok {
			continue
		}
		uniq[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func buildRegistryKey(component, name string) string {
	component = normalizeToken(component)
	if component != "" {
		return component
	}
	return normalizeToken(name)
}

func normalizeToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, ".")
	s = strings.TrimSuffix(s, ".")
	return s
}
