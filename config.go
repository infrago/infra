/*
	config 配置模块
	支持接入配置中心，远程拉取配置信息
	例如edcd, consul, nomad, redis等
	default为文件版配置，直接读当前目录或指定目录的文件
	支持格式 json, toml, yaml
*/

package infra

import (
	"errors"
	"sync"

	. "github.com/infrago/base"
)

var (
	ErrInvalidConfig = errors.New("Invalid config.")
)

var (
	infraConfig = &configModule{
		configurators: make(map[string]Configurator, 0),
	}
)

type (
	configModule struct {
		mutex         sync.Mutex
		configurators map[string]Configurator
	}
	configAction func(string) (Map, error)
	Configurator struct {
		// Name 名称
		Name string
		// Text 说明
		Text string
		// Alias 别名
		Alias []string
		// Action 获取配置
		Action configAction
	}
)

// Configurator 配置器
func (this *configModule) Configurator(name string, config Configurator) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if infra.override() {
			this.configurators[key] = config
		} else {
			if _, ok := this.configurators[key]; ok == false {
				this.configurators[key] = config
			}
		}
	}
}

func (this *configModule) Adsasdf(name string, value Any) {
	switch val := value.(type) {
	case Configurator:
		this.Configurator(name, val)
	}
}

func (this *configModule) Register(o Object) {
	switch val := o.Object.(type) {
	case Configurator:
		this.Configurator(o.Name, val)
	}
}

func (this *configModule) Configure(global Map) {
}

func (this *configModule) Initialize() {
}

func (this *configModule) Connect() {
}

func (this *configModule) Launch() {
}

func (this *configModule) Terminate() {
}
