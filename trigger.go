package infra

import (
	"errors"
	"sync"

	. "github.com/infrago/base"
)

const (
	START = "start"
	STOP  = "stop"
)

var (
	ErrInvalidTrigger = errors.New("Invalid trigger.")
)

var (
	infraTrigger = &triggerModule{
		config:   triggerConfig{},
		triggers: make(map[string][]Trigger, 0),
		methods:  make(map[string][]string, 0),
	}
)

type (
	triggerConfig struct {
	}

	Trigger struct {
		Name     string   `json:"name"`
		Text     string   `json:"desc"`
		Alias    []string `json:"-"`
		Nullable bool     `json:"null"`
		Args     Vars     `json:"args"`
		Data     Vars     `json:"data"`
		Setting  Map      `json:"-"`
		Coding   bool     `json:"-"`
		Action   Any      `json:"-"`
	}

	triggerModule struct {
		mutex  sync.Mutex
		config triggerConfig

		triggers map[string][]Trigger
		methods  map[string][]string
	}
)

// Register
func (this *triggerModule) Register(name string, value Any) {
	switch val := value.(type) {
	case Trigger:
		this.Trigger(name, val)
	}
}

// Configure
func (this *triggerModule) Configure(global Map) {
	// var config Map
	// if vv, ok := global["trigger"].(Map); ok {
	// 	config = vv
	// }

	// if secret, ok := config["secret"].(string); ok {
	// 	this.config.Secret = secret
	// }

	// //默认过期时间，单位秒
	// if expire, ok := config["expire"].(string); ok {
	// 	dur, err := util.ParseDuration(expire)
	// 	if err == nil {
	// 		this.config.Expire = dur
	// 	}
	// }
	// if expire, ok := config["expire"].(int); ok {
	// 	this.config.Expire = time.Second * time.Duration(expire)
	// }
	// if expire, ok := config["expire"].(int64); ok {
	// 	this.config.Expire = time.Second * time.Duration(expire)
	// }
	// if expire, ok := config["expire"].(float64); ok {
	// 	this.config.Expire = time.Second * time.Duration(expire)
	// }
}

func (this *triggerModule) Initialize() {

	for name, triggers := range this.triggers {
		if _, ok := this.methods[name]; ok == false {
			this.methods[name] = make([]string, 0)
		}
		for _, config := range triggers {
			randName := infraCodec.Generate()

			method := Method{
				config.Name, config.Text, config.Alias, config.Nullable,
				config.Args, config.Data, config.Setting, config.Coding, config.Action,
				false, false, _EMPTY,
			}
			infraEngine.Method(randName, method)

			//记录触发器
			this.methods[name] = append(this.methods[name], randName)
		}

	}
}
func (this *triggerModule) Connect() {
}
func (this *triggerModule) Launch() {
}
func (this *triggerModule) Terminate() {
}

// ----------------------
// 注意：这里mCodec还没初始化，所有无法生成ID
// 需要放到 init中去处理
func (this *triggerModule) Trigger(name string, config Trigger) {
	if _, ok := this.triggers[name]; ok == false {
		this.triggers[name] = make([]Trigger, 0)
	}

	//记录触发器
	this.triggers[name] = append(this.triggers[name], config)
}

// ------------------------- 方法 ----------------------------
// 触发
func (this *triggerModule) Toggle(name string, values ...Map) {
	if ms, ok := this.methods[name]; ok {
		for _, m := range ms {
			go Execute(m, values...)
		}
	}
}
func (this *triggerModule) SyncToggle(name string, values ...Map) {
	if ms, ok := this.methods[name]; ok {
		for _, m := range ms {
			Execute(m, values...)
		}
	}
}

func Toggle(name string, values ...Map) {
	infraTrigger.Toggle(name, values...)
}

func SyncToggle(name string, values ...Map) {
	infraTrigger.SyncToggle(name, values...)
}
