package bamgoo

import (
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	. "github.com/bamgoo/base"
)

const defaultCallTimeout = 5 * time.Second

var core = &coreModule{
	entries: make(map[string]coreEntry, 0),
}

type (
	coreModule struct {
		mutex   sync.RWMutex
		entries map[string]coreEntry
	}
	coreEntry struct {
		remote bool
		kind   string
		span   string
		target string

		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Data     Vars
		Action   Any
		Setting  Map
	}
)

const (
	coreKindMethod  = "method"
	coreKindService = "service"
	coreKindTrigger = "trigger"
)

type (
	Methods map[string]Method
	Method  struct {
		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Data     Vars
		Action   Any
		Setting  Map
	}
	Services map[string]Service
	Service  struct {
		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Data     Vars
		Action   Any
		Setting  Map
	}
)

func (e *coreModule) Register(name string, value Any) {
	switch v := value.(type) {
	case Method:
		e.RegisterMethod(name, v)
	case Service:
		e.RegisterService(name, v)
	case Methods:
		e.RegisterMethods(name, v)
	case Services:
		e.RegisterServices(name, v)
	}
}

func (e *coreModule) RegisterMethods(prefix string, methods Methods) {
	for key, method := range methods {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		e.RegisterMethod(name, method)
	}
}

func (e *coreModule) RegisterServices(prefix string, services Services) {
	for key, service := range services {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		e.RegisterService(name, service)
	}
}

func (e *coreModule) RegisterMethod(name string, method Method) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if name == "" {
		return
	}

	if _, ok := e.entries[name]; ok {
		panic("method already registered: " + name)
	}

	e.entries[name] = coreEntry{
		remote:   false,
		kind:     coreKindMethod,
		span:     "method:" + name,
		target:   name,
		Name:     name,
		Desc:     method.Desc,
		Nullable: method.Nullable,
		Args:     method.Args,
		Data:     method.Data,
		Action:   method.Action,
		Setting:  method.Setting,
	}
}

func (e *coreModule) RegisterService(name string, service Service) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if name == "" {
		return
	}
	if _, ok := e.entries[name]; ok {
		panic("service already registered: " + name)
	}
	e.entries[name] = coreEntry{
		remote:   true,
		kind:     coreKindService,
		span:     "service:" + name,
		target:   name,
		Name:     name,
		Desc:     service.Desc,
		Nullable: service.Nullable,
		Args:     service.Args,
		Data:     service.Data,
		Action:   service.Action,
	}
}

func (e *coreModule) registerTriggerMethod(methodName, triggerName string, cfg Trigger) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if methodName == "" {
		return
	}
	if _, ok := e.entries[methodName]; ok {
		panic("trigger already registered: " + methodName)
	}

	action := cfg.Action
	e.entries[methodName] = coreEntry{
		remote:   false,
		kind:     coreKindTrigger,
		span:     "trigger:" + triggerName,
		target:   triggerName,
		Name:     cfg.Name,
		Desc:     cfg.Desc,
		Nullable: cfg.Nullable,
		Args:     cfg.Args,
		Data:     cfg.Data,
		Action: func(ctx *Context) {
			if action != nil {
				action(ctx)
			}
		},
	}
}

func (e *coreModule) Config(Map) {}
func (e *coreModule) Setup()     {}
func (e *coreModule) Open()      {}
func (e *coreModule) Start()     {}
func (e *coreModule) Stop()      {}
func (e *coreModule) Close()     {}

func (e *coreModule) Wait() {
	// 待处理，加入自己的退出信号
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// Invoke calls a method/service (local first, then remote via bus).
func (e *coreModule) Invoke(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	if meta == nil {
		meta = NewMeta()
	}

	spanName := name
	target := name
	entryKind := ""
	e.mutex.RLock()
	if entry, ok := e.entries[name]; ok {
		if entry.span != "" {
			spanName = entry.span
		}
		if entry.target != "" {
			target = entry.target
		}
		entryKind = entry.kind
	}
	e.mutex.RUnlock()

	span := meta.Begin(spanName, TraceAttrs("bamgoo", TraceKindInternal, target, Map{
		"module":    "core",
		"operation": "invoke",
		"entry":     entryKind,
	}))

	if data, res, ok := e.invokeLocal(meta, name, value, settings...); ok {
		if res != nil && res.Fail() {
			span.End(errors.New(res.Error()))
		} else {
			span.End()
		}
		return data, res
	}
	data, res := e.invokeRemote(meta, name, value)
	if res != nil && res.Fail() {
		span.End(errors.New(res.Error()))
	} else {
		span.End()
	}
	return data, res
}

// localInvoke only calls local method/service, does not go through bus.
// Returns (data, res, found) where found indicates if local entry exists.
func (e *coreModule) invokeLocal(meta *Meta, name string, value Map, settings ...Map) (Map, Res, bool) {
	e.mutex.RLock()
	entry, ok := e.entries[name]
	e.mutex.RUnlock()

	if !ok || entry.Action == nil {
		return nil, nil, false
	}

	if meta == nil {
		meta = NewMeta()
	}
	ctx := &Context{
		Meta:    meta,
		Name:    name,
		Config:  &entry,
		Setting: Map{},
		Value:   value,
		Args:    cloneMap(value),
	}
	if len(entry.Args) > 0 {
		args := Map{}
		res := Mapping(entry.Args, value, args, false, false, ctx.Timezone())
		if res != nil && res.Fail() {
			return nil, res, true
		}
		ctx.Args = args
	}
	for k, v := range entry.Setting {
		ctx.Setting[k] = v
	}
	for _, setting := range settings {
		if setting == nil {
			continue
		}
		for k, v := range setting {
			ctx.Setting[k] = v
		}
	}
	data, res := invokeAction(entry.Action, ctx)
	if len(entry.Data) > 0 && (res == nil || !res.Fail()) && data != nil {
		mapped := Map{}
		mappedRes := Mapping(entry.Data, data, mapped, true, false, ctx.Timezone())
		if mappedRes != nil && mappedRes.Fail() {
			return nil, mappedRes, true
		}
		data = mapped
	}
	return data, res, true
}

// remoteInvoke calls remote service via bus.
func (e *coreModule) invokeRemote(meta *Meta, name string, value Map) (Map, Res) {
	if meta == nil {
		meta = NewMeta()
	}
	return hook.Request(meta, name, value, defaultCallTimeout)
}

func cloneMap(in Map) Map {
	out := Map{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func invokeAction(action Any, ctx *Context) (Map, Res) {
	switch fn := action.(type) {
	case func(*Context):
		fn(ctx)
		return Map{}, OK
	case func(*Context) Map:
		return fn(ctx), OK
	case func(*Context) Res:
		return Map{}, defaultResult(fn(ctx))
	case func(*Context) (Map, Res):
		data, res := fn(ctx)
		return data, defaultResult(res)
	default:
		return nil, Fail.With("invalid action signature")
	}
}

func defaultResult(res Res) Res {
	if res == nil {
		return OK
	}
	return res
}
