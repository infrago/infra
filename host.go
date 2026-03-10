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
		InvokeLocalMethod(meta *Meta, name string, value Map) (Map, Res, bool)
		InvokeLocalService(meta *Meta, name string, value Map, settings ...Map) (Map, Res, bool)
		InvokeLocalMessage(meta *Meta, name string, value Map) (Map, Res, bool)
		RegisterLocal(name string, value Any)
	}
)

func (h *infragoHost) InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocal(meta, name, value)
}

func (h *infragoHost) InvokeLocalMethod(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocalWithKinds(meta, name, value, []string{coreKindMethod})
}

func (h *infragoHost) InvokeLocalService(meta *Meta, name string, value Map, settings ...Map) (Map, Res, bool) {
	return core.invokeLocalWithKinds(meta, name, value, []string{coreKindService}, settings...)
}

func (h *infragoHost) InvokeLocalMessage(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocalWithKinds(meta, name, value, []string{coreKindMessage})
}

func (h *infragoHost) RegisterLocal(name string, value Any) {
	infrago.Register(name, value)
}
