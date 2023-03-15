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
	mConfig = &configModule{
		config: configConfig{DEFAULT, Map{}},
	}

	errInvalidConfig = errors.New("Invalid config.")
)

type (
	configConfig struct {
		Driver  string
		Setting Map
	}
	configModule struct {
		mutex  sync.Mutex
		config configConfig
	}
)

func (this *configModule) Register(name string, value Any) {
}

func (this *configModule) Configure(global Map) {
}

func (this *configModule) Initialize() {
	if this.config.Driver == "" {
		this.config.Driver = DEFAULT
	}
}

func (this *configModule) Connect() {
}

func (this *configModule) Launch() {
}

func (this *configModule) Terminate() {
}
