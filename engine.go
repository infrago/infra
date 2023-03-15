package infra

import (
	"sync"
	"time"

	. "github.com/infrago/base"
)

var (
	infraEngine = newEngineModule()
)

func newEngineModule() *engineModule {
	return &engineModule{
		methods: make(map[string]Method, 0),
	}
}

const (
	engineInvoke   = "invoke"
	engineInvokes  = "invokes"
	engineInvoking = "invoking"
	engineInvoked  = "invoked"
	engineInvokee  = "invokee"
	engineInvoker  = "invoker"
)

type (
	Method struct {
		Name     string   `json:"name"`
		Text     string   `json:"desc"`
		Alias    []string `json:"-"`
		Nullable bool     `json:"null"`
		Args     Vars     `json:"args"`
		Data     Vars     `json:"data"`
		Setting  Map      `json:"-"`
		Coding   bool     `json:"-"`
		Action   Any      `json:"-"`

		Sign bool   `json:"sign"`
		Auth bool   `json:"auth"`
		Kind string `json:"kind"`
	}

	Service struct {
		Name     string   `json:"name"`
		Text     string   `json:"desc"`
		Alias    []string `json:"-"`
		Nullable bool     `json:"null"`
		Args     Vars     `json:"args"`
		Data     Vars     `json:"data"`
		Setting  Map      `json:"-"`
		Coding   bool     `json:"-"`
		Action   Any      `json:"-"`

		Sign bool   `json:"token"`
		Auth bool   `json:"auth"`
		Kind string `json:"kind"`
	}

	Context struct {
		*Meta
		Name    string
		Config  Method
		Setting Map

		Value Map
		Args  Map
	}

	Lib struct {
		meta   *Meta
		engine *engineModule

		Name    string
		Setting Map
	}

	engineModule struct {
		mutex   sync.Mutex
		methods map[string]Method
	}
)

// Register
func (module *engineModule) Register(key string, value Any) {
	switch val := value.(type) {
	case Method:
		module.Method(key, val)
	case Service:
		module.Service(key, val)
	}
}

// Configure
func (module *engineModule) Configure(value Map) {
}

// Initialize
func (module *engineModule) Initialize() {
}

// Connect
func (module *engineModule) Connect() {
}

// Launch
func (module *engineModule) Launch() {
}

// Terminate
func (module *engineModule) Terminate() {
}

func (module *engineModule) Method(name string, config Method) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if _, ok := module.methods[key]; ok == false {
			module.methods[key] = config
		} else {
			panic("[engine]Method已经存在了")
		}
	}
}

func (module *engineModule) Service(name string, config Service) {
	method := Method{
		config.Name, config.Text, config.Alias, config.Nullable,
		config.Args, config.Data, config.Setting, config.Coding, config.Action,
		config.Sign, config.Auth, config.Kind,
	}
	module.Method(name, method)
}

// 给本地 invoke 的，加上远程调用
func (module *engineModule) Call(meta *Meta, name string, value Map, settings ...Map) (Map, Res, string) {
	data, callRes, tttt := module.call(meta, name, value, settings...)
	// 待优化，返回包装类型，好在远程调用后处理data
	// 暂时不处理data，整个返回

	if callRes == Nothing {
		// //本地不存在的时候，去总线请求
		echo, err := infraBridge.Request(meta, name, value, time.Second*5)
		if err != nil {
			return nil, errorResult(err), tttt
		}

		tttt = echo.Type

		//处理返回数据的包装
		if echo.Data != nil && echo.Args != nil {

			var args Vars
			if tttt == engineInvokes {
				args = invokesDataConfig(echo.Args)
			} else if tttt == engineInvoking {
				args = invokingDataConfig(echo.Args)
			} else if tttt == engineInvokee {
				args = invokeeDataConfig()
			} else if tttt == engineInvoker {
				args = invokerDataConfig()
			} else if tttt == engineInvoked {
				//使用返果来判定
			} else {
				//默认不处理
			}

			if args != nil {
				value := Map{}
				mRes := infraBasic.Mapping(args, echo.Data, value, false, false, meta.Timezone())
				if mRes == nil && mRes.OK() {
					echo.Data = value
				}
			}
		}

		// //直接返因，因为mBus.Request已经处理args和data
		return echo.Data, newResult(echo.Code, echo.Text), echo.Type

		// return data, callRes, tttt

	} else {
		return data, callRes, tttt
	}
}

// 真实的方法调用，纯本地调用
// 此方法不能远程调用，要不然就死循环了
func (module *engineModule) call(meta *Meta, name string, value Map, settings ...Map) (Map, Res, string) {
	tttt := engineInvoke
	if _, ok := module.methods[name]; ok == false {
		return nil, Nothing, tttt
	}

	config := module.methods[name]

	if meta == nil {
		meta = &Meta{name: name, payload: value}
		defer meta.close()
	}

	ctx := &Context{Meta: meta}
	ctx.Name = name
	ctx.Config = config
	ctx.Setting = Map{}

	for k, v := range config.Setting {
		ctx.Setting[k] = v
	}
	if len(settings) > 0 {
		for k, v := range settings[0] {
			ctx.Setting[k] = v
		}
	}

	if value == nil {
		value = Map{}
	}

	args := Map{}
	if config.Args != nil {
		res := infraBasic.Mapping(config.Args, value, args, config.Nullable, false, ctx.Timezone())
		if res != nil && res.Fail() {
			return nil, res, tttt
		}
	}

	ctx.Value = value
	ctx.Args = args

	// process := &Process{
	// 	context: ctx, engine: module,
	// 	Name: name, Config: config, Setting: setting,
	// 	Value: value, Args: args,
	// }

	data := Map{}
	result := OK //默认为成功

	switch ff := config.Action.(type) {
	case func(*Context):
		ff(ctx)
	case func(*Context) Res:
		result = ff(ctx)
		//查询是否
	case func(*Context) bool:
		ok := ff(ctx)
		if ok {
			result = OK
		} else {
			result = Fail
		}
		//查询单个
	case func(*Context) Map:
		data = ff(ctx)
	case func(*Context) (Map, Res):
		data, result = ff(ctx)

		//查询列表
	case func(*Context) []Map:
		items := ff(ctx)
		data = Map{"items": items}
		tttt = engineInvokes
	case func(*Context) ([]Map, Res):
		items, res := ff(ctx)
		data = Map{"items": items}
		result = res
		tttt = engineInvokes

		//统计的玩法
	case func(*Context) int:
		count := ff(ctx)
		data = Map{"count": float64(count)}
		tttt = engineInvokee
	case func(*Context) int64:
		count := ff(ctx)
		data = Map{"count": float64(count)}
		tttt = engineInvokee
	case func(*Context) float64:
		count := ff(ctx)
		data = Map{"count": count}
		tttt = engineInvokee

		//查询分页的玩法
	case func(*Context) ([]Map, int64):
		items, count := ff(ctx)
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) ([]Map, int64, Res):
		items, count, res := ff(ctx)
		result = res
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) (int64, []Map):
		count, items := ff(ctx)
		data = Map{"count": count, "items": items}
		tttt = engineInvoking
	case func(*Context) (int64, []Map, Res):
		count, items, res := ff(ctx)
		result = res
		data = Map{"count": count, "items": items}
		tttt = engineInvoking

	case func(*Context) (Map, []Map):
		item, items := ff(ctx)
		data = Map{"item": item, "items": items}
		tttt = engineInvoker
	case func(*Context) (Map, []Map, Res):
		item, items, res := ff(ctx)
		result = res
		data = Map{"item": item, "items": items}
		tttt = engineInvoker
	}

	//参数解析
	if config.Data != nil {
		out := Map{}
		err := infraBasic.Mapping(config.Data, data, out, false, false, ctx.Timezone())
		if err == nil || err.OK() {
			return out, result, tttt
		}
	}

	//参数如果解析失败，就原版返回
	return data, result, tttt
}

func (module *engineModule) Execute(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	m, r, _ := module.call(meta, name, value, settings...)
	return m, r
}

// 以下几个方法要做些交叉处理
func (module *engineModule) Invoke(meta *Meta, name string, value Map, settings ...Map) (Map, Res) {
	data, res, tttt := module.Call(meta, name, value, settings...)

	if tttt == engineInvokes {
		if vvs, ok := data["items"].([]Map); ok && len(vvs) > 0 {
			return vvs[0], res
		}
	}

	return data, res
}

func (module *engineModule) Invokes(meta *Meta, name string, value Map, settings ...Map) ([]Map, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)

	if data != nil {

		// if name == "share.GetTerminalHolders" {
		// 	fmt.Println("eng.Invokes", tttt, data)
		// }

		if results, ok := data["items"].([]Map); ok {
			return results, res
		} else if results, ok := data["items"].([]Any); ok {
			items := []Map{}
			for _, result := range results {
				if item, ok := result.(Map); ok {
					items = append(items, item)
				}
			}
			return items, res
		} else {
			hasKey := false
			for range data {
				hasKey = true
			}

			if hasKey {
				return []Map{data}, res
			} else {
				return []Map{}, res
			}
		}
	}

	return []Map{}, res
}
func (module *engineModule) Invoked(meta *Meta, name string, value Map, settings ...Map) (bool, Res) {
	_, res, _ := module.Call(meta, name, value, settings...)
	if res == nil || res.OK() {
		return true, res
	}
	return false, res
}
func (module *engineModule) Invoking(meta *Meta, name string, offset, limit int64, value Map, settings ...Map) (int64, []Map, Res) {
	if value == nil {
		value = Map{}
	}
	value["offset"] = offset
	value["limit"] = limit

	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return 0, nil, res
	}

	count, countOK := data["count"].(int64)
	items, itemsOK := data["items"].([]Map)
	if countOK && itemsOK {
		return count, items, res
	}

	return 0, []Map{data}, res
}

func (module *engineModule) Invoker(meta *Meta, name string, value Map, settings ...Map) (Map, []Map, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return nil, nil, res
	}

	item, itemOK := data["item"].(Map)
	items, itemsOK := data["items"].([]Map)

	if itemOK && itemsOK {
		return item, items, res
	}

	return data, []Map{data}, res
}

func (module *engineModule) Invokee(meta *Meta, name string, value Map, settings ...Map) (float64, Res) {
	data, res, _ := module.Call(meta, name, value, settings...)
	if res != nil && res.Fail() {
		return 0, res
	}

	if vv, ok := data["count"].(float64); ok {
		return vv, res
	} else if vv, ok := data["count"].(int64); ok {
		return float64(vv), res
	}

	return 0, res
}

func (module *engineModule) Library(meta *Meta, name string, settings ...Map) *Lib {
	setting := make(Map)
	for _, sets := range settings {
		for k, v := range sets {
			setting[k] = v
		}
	}
	return &Lib{meta, module, name, setting}
}

// 获取参数定义
// 支持远程获取
// 待优化
func (module *engineModule) Arguments(name string, extends ...Vars) Vars {
	args := Vars{}

	if config, ok := module.methods[name]; ok {
		for k, v := range config.Args {
			args[k] = v
		}
	} else {

		//去集群找定义，待处理

		//停用，因为注册路由的时候，集群还没有初始化，自然拿不到定义
		// vvv, err := module.infra.Cluster.arguments(name)
		// if err == nil {
		// 	args = vvv
		// }
	}

	return VarsExtend(args, extends...)
}

// ------- logic 方法 -------------
func (lgc *Lib) naming(name string) string {
	return lgc.Name + "." + name
}

func (lgc *Lib) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := lgc.engine.Invoke(lgc.meta, lgc.naming(name), value, lgc.Setting)
	lgc.meta.result = res
	return vvv
}

func (logic *Lib) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := logic.engine.Invokes(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return vvs
}
func (logic *Lib) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := logic.engine.Invoked(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return vvv
}
func (logic *Lib) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := logic.engine.Invoking(logic.meta, logic.naming(name), offset, limit, value, logic.Setting)
	logic.meta.result = res
	return count, items
}

// gob之后，不需要再定义data模型
func (logic *Lib) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := logic.engine.Invoker(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return item, items
}

func (logic *Lib) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := logic.engine.Invokee(logic.meta, logic.naming(name), value, logic.Setting)
	logic.meta.result = res
	return count
}

//---------------------------- engine config data

func invokingArgsConfig(offset, limit int64, extends ...Vars) Vars {
	config := Vars{
		"offset": Var{
			Type: "int", Required: true, Default: offset, Name: "offset", Text: "offset",
		},
		"limit": Var{
			Type: "int", Required: true, Default: limit, Name: "limit", Text: "limit",
		},
	}

	return VarsExtend(config, extends...)
}
func invokingDataConfig(childrens ...Vars) Vars {
	var children Vars
	if len(childrens) > 0 {
		children = childrens[0]
	}
	config := Vars{
		"count": Var{
			Type: "int", Required: true, Default: 0, Name: "统计数", Text: "统计数",
		},
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			Children: children,
		},
	}
	return config
}

func invokesDataConfig(childrens ...Vars) Vars {
	var children Vars
	if len(childrens) > 0 {
		children = childrens[0]
	}
	config := Vars{
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			Children: children,
		},
	}
	return config
}

func invokeeDataConfig() Vars {
	config := Vars{
		"count": Var{
			Type: "float", Required: true, Default: 0, Name: "统计数", Text: "统计数",
		},
	}
	return config
}

// 待处理，返回模型有点不好定义
func invokerDataConfig() Vars {
	config := Vars{
		"item": Var{
			Type: "json", Required: true, Name: "数据", Text: "数据",
			// Children: children,
		},
		"items": Var{
			Type: "[json]", Required: true, Name: "数据列表", Text: "数据列表",
			// Children: children,
		},
	}
	return config
}

//-------------------------------------------------------------------------------------------------------

// 方法参数
func Arguments(name string, extends ...Vars) Vars {
	return infraEngine.Arguments(name, extends...)
}

// 直接执行，同步，本地
func Execute(name string, values ...Any) (Map, Res) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	return infraEngine.Execute(nil, name, value)
}

// 原始调用，给BUS总线使用
func Calling(meta *Meta, name string, value Map, settings ...Map) (Map, Res, string) {
	return infraEngine.call(meta, name, value, settings...)
}
