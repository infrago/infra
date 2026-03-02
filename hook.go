package infra

import (
	"errors"
	"sync"
	"time"

	base "github.com/infrago/base"
)

var (
	errBusHookMissing    = errors.New("bus hook not registered")
	errConfigHookMissing = errors.New("config hook not registered")
	errTokenHookMissing  = errors.New("token hook not registered")
)

// Hook exposes hook registrations and access (main -> sub).
var hook = &infragoHook{}

type (
	TraceSpan interface {
		End(...base.Any)
	}

	infragoHook struct {
		mutex sync.RWMutex

		bus    BusHook
		config ConfigHook
		trace  TraceHook
		token  TokenHook
	}

	BusHook interface {
		Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res)
		Broadcast(meta *Meta, name string, value base.Map) error
		Publish(meta *Meta, name string, value base.Map) error
		Enqueue(meta *Meta, name string, value base.Map) error
		Stats() []ServiceStats
		ListNodes() []NodeInfo
		ListServices() []ServiceInfo
	}

	ConfigHook interface {
		LoadConfig() (base.Map, error)
	}

	TraceHook interface {
		Begin(meta *Meta, name string, attrs base.Map) TraceSpan
		Trace(meta *Meta, name string, status string, attrs base.Map) error
	}

	TokenHook interface {
		Sign(meta *Meta, req TokenSignRequest) (TokenSession, error)
		Verify(meta *Meta, token string) (TokenSession, error)
		RevokeToken(meta *Meta, token string, expires int64) error
		RevokeTokenID(meta *Meta, tokenID string, expires int64) error
	}
)

// Attach dispatches Module.Attach based on type.
func (h *infragoHook) Attach(value base.Any) {
	switch v := value.(type) {
	case BusHook:
		h.AttachBus(v)
	case ConfigHook:
		h.AttachConfig(v)
	case TraceHook:
		h.AttachTrace(v)
	case TokenHook:
		h.AttachToken(v)
	}
}

func (h *infragoHook) AttachBus(hook BusHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid bus hook")
	}

	h.bus = hook
}

func (h *infragoHook) AttachConfig(hook ConfigHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid config hook")
	}

	h.config = hook
}

func (h *infragoHook) AttachTrace(hook TraceHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid trace hook")
	}

	h.trace = hook
}

func (h *infragoHook) AttachToken(hook TokenHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid token hook")
	}

	h.token = hook
}

func (h *infragoHook) LoadConfig() (base.Map, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.config == nil {
		return nil, errConfigHookMissing
	}
	return h.config.LoadConfig()
}

// Request sends a bus request (main -> sub).
func (h *infragoHook) Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil, ErrorResult(errBusHookMissing)
	}
	return h.bus.Request(meta, name, value, timeout)
}

func (h *infragoHook) Broadcast(name string, value base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return errBusHookMissing
	}
	return h.bus.Broadcast(nil, name, value)
}

func (h *infragoHook) Publish(name string, value base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return errBusHookMissing
	}
	return h.bus.Publish(nil, name, value)
}

func (h *infragoHook) Enqueue(name string, value base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return errBusHookMissing
	}
	return h.bus.Enqueue(nil, name, value)
}

func (h *infragoHook) Stats() []ServiceStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil
	}
	return h.bus.Stats()
}

func (h *infragoHook) ListNodes() []NodeInfo {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil
	}
	return h.bus.ListNodes()
}

func (h *infragoHook) ListServices() []ServiceInfo {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil
	}
	return h.bus.ListServices()
}

func (h *infragoHook) Begin(meta *Meta, name string, attrs base.Map) TraceSpan {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.trace == nil {
		return noopTraceSpan{}
	}
	return h.trace.Begin(meta, name, attrs)
}

func (h *infragoHook) Trace(meta *Meta, name string, status string, attrs base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.trace == nil {
		return nil
	}
	return h.trace.Trace(meta, name, status, attrs)
}

func (h *infragoHook) SignToken(meta *Meta, req TokenSignRequest) (TokenSession, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.token == nil {
		return TokenSession{}, errTokenHookMissing
	}
	return h.token.Sign(meta, req)
}

func (h *infragoHook) VerifyToken(meta *Meta, token string) (TokenSession, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.token == nil {
		return TokenSession{}, errTokenHookMissing
	}
	return h.token.Verify(meta, token)
}

func (h *infragoHook) RevokeToken(meta *Meta, token string, expires int64) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.token == nil {
		return errTokenHookMissing
	}
	return h.token.RevokeToken(meta, token, expires)
}

func (h *infragoHook) RevokeTokenID(meta *Meta, tokenID string, expires int64) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.token == nil {
		return errTokenHookMissing
	}
	return h.token.RevokeTokenID(meta, tokenID, expires)
}

type noopTraceSpan struct{}

func (noopTraceSpan) End(...base.Any) {}
