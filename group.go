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

func (this *group) Name() string {
	return this.name
}
func (this *group) Register(args ...Any) {
	values := make([]Any, 0)
	for _, arg := range args {
		if ss, ok := arg.(string); ok {
			prefix := this.name + "."
			if strings.HasPrefix(ss, ".") {
				prefix = this.name
			}
			values = append(values, prefix+ss)
		} else {
			values = append(values, arg)
		}
	}

	Register(values...)
}

func (this *group) Result(ok bool, state string, text string) Res {
	code := 0
	if ok == false {
		code = this.base
		this.base++
	}

	if this.name != "" && !strings.HasPrefix(state, this.name+".") {
		state = this.name + "." + state
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
