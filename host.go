package bamgoo

import (
	. "github.com/bamgoo/base"
)

var host = &bamgooHost{}

type (
	bamgooHost struct {
	}

	Host interface {
		InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool)
		RegisterLocal(name string, value Any)
	}
)

func (h *bamgooHost) InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocal(meta, name, value)
}

func (h *bamgooHost) RegisterLocal(name string, value Any) {
	bamgoo.Register(name, value)
}
