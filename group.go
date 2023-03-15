package infra

import (
	"strings"

	. "github.com/infrago/base"
)

type (
	group struct {
		name string
		base int
	}
)

//------------ group ----------------

func (lib *group) Name() string {
	return lib.name
}
func (lib *group) Register(name string, value Any) {
	//20221230全部加前缀，不管是不是头重复
	// if lib.name != "" && !strings.HasPrefix(name, prefix) {
	if lib.name != "" {
		prefix := lib.name + "."
		if strings.HasPrefix(name, ".") {
			prefix = lib.name
		}

		name = prefix + name
	}

	args := make([]Any, 0)
	args = append(args, name, value)

	Register(args...)
}

func (lib *group) Result(ok bool, state string, text string) Res {
	code := 0
	if ok == false {
		code = lib.base
		lib.base++
	}

	if lib.name != "" && !strings.HasPrefix(state, lib.name+".") {
		state = lib.name + "." + state
	}
	return Result(code, state, text)
}

func Module(name string, bases ...int) *group {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &group{name, base}
}

func Library(name string, bases ...int) *group {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &group{name, base}
}

func Site(name string, bases ...int) *group {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &group{name, base}
}

func Group(name string, bases ...int) *group {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &group{name, base}
}
