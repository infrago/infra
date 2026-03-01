package infra

import (
	. "github.com/infrago/base"
)

var host = &infragoHost{}

type (
	infragoHost struct {
	}

	Host interface {
		InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool)
		RegisterLocal(name string, value Any)
	}
)

func (h *infragoHost) InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocal(meta, name, value)
}

func (h *infragoHost) RegisterLocal(name string, value Any) {
	infra.Register(name, value)
}
