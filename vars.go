package infra

import . "github.com/infrago/base"

func extendVars(config Vars, extends ...Vars) Vars {
	if len(extends) == 0 {
		return config
	}

	for key, val := range extends[0] {
		if val.Nil() {
			delete(config, key)
			continue
		}
		config[key] = val
	}

	return config
}
