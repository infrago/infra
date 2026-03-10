package infra

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	. "github.com/infrago/base"
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
		retry  []time.Duration

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
	coreKindMessage = "message"
	coreKindTrigger = "trigger"

	dispatchAttemptSetting = "_dispatch_attempt"
	dispatchFinalSetting   = "_dispatch_final"
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
		Retry    []time.Duration
		Setting  Map
	}
	Messages map[string]Message
	Message  struct {
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
	case Message:
		e.RegisterMessage(name, v)
	case Methods:
		e.RegisterMethods(name, v)
	case Services:
		e.RegisterServices(name, v)
	case Messages:
		e.RegisterMessages(name, v)
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

func (e *coreModule) RegisterMessages(prefix string, messages Messages) {
	for key, message := range messages {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		e.RegisterMessage(name, message)
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
		retry:    cloneDurations(service.Retry),
		Name:     name,
		Desc:     service.Desc,
		Nullable: service.Nullable,
		Args:     service.Args,
		Data:     service.Data,
		Action:   service.Action,
		Setting:  service.Setting,
	}
}

func (e *coreModule) RegisterMessage(name string, message Message) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if name == "" {
		return
	}
	if _, ok := e.entries[name]; ok {
		panic("message already registered: " + name)
	}
	e.entries[name] = coreEntry{
		remote:   false,
		kind:     coreKindMessage,
		span:     "message:" + name,
		target:   name,
		Name:     name,
		Desc:     message.Desc,
		Nullable: message.Nullable,
		Args:     message.Args,
		Data:     message.Data,
		Action:   message.Action,
		Setting:  message.Setting,
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

func (e *coreModule) dispatchRetries(name string) []time.Duration {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	entry, ok := e.entries[name]
	if !ok || entry.kind != coreKindService || len(entry.retry) == 0 {
		return nil
	}
	return cloneDurations(entry.retry)
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

func (e *coreModule) Arguments(name string, extends ...Vars) Vars {
	e.mutex.RLock()
	entry, ok := e.entries[name]
	e.mutex.RUnlock()

	args := Vars{}
	if ok {
		for key, val := range entry.Args {
			args[key] = val
		}
	}

	return extendVars(args, extends...)
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

	span := meta.Begin(spanName, TraceAttrs("infrago", entryKind, target, Map{
		"module":    "core",
		"operation": "invoke",
	}))

	if data, res, ok := e.invokeLocal(meta, name, value, settings...); ok {
		if res != nil && res.Fail() {
			span.End(res)
		} else {
			span.End()
		}
		return data, res
	}
	data, res := e.invokeRemote(meta, name, value)
	if res != nil && res.Fail() {
		span.End(res)
	} else {
		span.End()
	}
	return data, res
}

// Execute calls only local method, and never falls back to remote bus.
func (e *coreModule) Execute(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	if meta == nil {
		meta = NewMeta()
	}
	span := meta.Begin("method:"+name, TraceAttrs("infrago", coreKindMethod, name, Map{
		"module":    "core",
		"operation": "execute",
	}))
	data, res, ok := e.invokeLocalWithKinds(meta, name, value, []string{coreKindMethod}, settings...)
	if !ok {
		res = Fail.With("method not found: " + name)
	}
	if res != nil && res.Fail() {
		span.End(res)
	} else {
		span.End()
	}
	return data, res
}

// Request always calls remote service through bus, regardless of local entries.
func (e *coreModule) Request(meta *Meta, name string, value Map, timeout ...time.Duration) (Map, Res) {
	if meta == nil {
		meta = NewMeta()
	}
	span := meta.Begin("service:"+name, TraceAttrs("infrago", coreKindService, name, Map{
		"module":    "core",
		"operation": "request",
	}))
	waitTimeout := defaultCallTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		waitTimeout = timeout[0]
	}
	data, res := hook.Request(meta, name, value, waitTimeout)
	if res != nil && res.Fail() {
		span.End(res)
	} else {
		span.End()
	}
	return data, res
}

// localInvoke only calls local method/service, does not go through bus.
// Returns (data, res, found) where found indicates if local entry exists.
func (e *coreModule) invokeLocal(meta *Meta, name string, value Map, settings ...Map) (Map, Res, bool) {
	return e.invokeLocalWithKinds(meta, name, value, nil, settings...)
}

func (e *coreModule) invokeLocalWithKinds(meta *Meta, name string, value Map, kinds []string, settings ...Map) (Map, Res, bool) {
	e.mutex.RLock()
	entry, ok := e.entries[name]
	e.mutex.RUnlock()

	if !ok || entry.Action == nil {
		return nil, nil, false
	}
	if len(kinds) > 0 && !containsString(kinds, entry.kind) {
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
	ctx.attempt = coreSettingInt(ctx.Setting[dispatchAttemptSetting], 1)
	ctx.final = coreSettingBool(ctx.Setting[dispatchFinalSetting], false)
	delete(ctx.Setting, dispatchAttemptSetting)
	delete(ctx.Setting, dispatchFinalSetting)

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
	case func(*Context) []Map:
		return Map{"items": fn(ctx)}, OK
	case func(*Context) ([]Map, Res):
		items, res := fn(ctx)
		return Map{"items": items}, defaultResult(res)
	case func(*Context) bool:
		return Map{"ok": fn(ctx)}, OK
	case func(*Context) Res:
		return Map{}, defaultResult(fn(ctx))
	case func(*Context) error:
		if err := fn(ctx); err != nil {
			return nil, Fail.With(err.Error())
		}
		return Map{}, OK
	case func(*Context) Any:
		return normalizeActionData(fn(ctx)), OK
	case func(*Context) (Any, Res):
		data, res := fn(ctx)
		return normalizeActionData(data), defaultResult(res)
	case func(*Context) (Map, Res):
		data, res := fn(ctx)
		return data, defaultResult(res)
	case func(*Context) (Map, []Map):
		data, items := fn(ctx)
		return packInvokeListData(data, items), OK
	case func(*Context) (Map, []Map, Res):
		data, items, res := fn(ctx)
		return packInvokeListData(data, items), defaultResult(res)
	case func(*Context) (int64, []Map):
		total, items := fn(ctx)
		return Map{
			"total": total,
			"items": items,
		}, OK
	case func(*Context) (int64, []Map, Res):
		total, items, res := fn(ctx)
		return Map{
			"total": total,
			"items": items,
		}, defaultResult(res)
	default:
		return nil, Fail.With("invalid action signature")
	}
}

func packInvokeListData(item Map, items []Map) Map {
	return Map{
		"item":  item,
		"items": items,
	}
}

func normalizeActionData(data Any) Map {
	switch v := data.(type) {
	case nil:
		return nil
	case Map:
		return v
	case []Map:
		return Map{"items": v}
	case bool:
		return Map{"ok": v}
	default:
		return Map{"value": v}
	}
}

func defaultResult(res Res) Res {
	if res == nil {
		return OK
	}
	return res
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func cloneDurations(in []time.Duration) []time.Duration {
	if len(in) == 0 {
		return nil
	}
	out := make([]time.Duration, 0, len(in))
	for _, item := range in {
		if item > 0 {
			out = append(out, item)
		}
	}
	return out
}

func coreSettingInt(v Any, fallback int) int {
	switch vv := v.(type) {
	case int:
		if vv > 0 {
			return vv
		}
	case int64:
		if vv > 0 {
			return int(vv)
		}
	case float64:
		if vv > 0 {
			return int(vv)
		}
	}
	return fallback
}

func coreSettingBool(v Any, fallback bool) bool {
	switch vv := v.(type) {
	case bool:
		return vv
	}
	return fallback
}
