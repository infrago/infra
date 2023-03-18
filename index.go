package infra

import (
	. "github.com/infrago/base"
)

func init() {
	Register(infraBridge)
	Register(infraBasic)
	Register(infraCodec)
	Register(infraEngine)
	Register(infraTrigger)
	Register(infraToken)

	builtin()
}

// Override
func Override(args ...bool) bool {
	return infra.override(args...)
}

// Register 注册各种内容
func Register(cfgs ...Any) {
	infra.register(cfgs...)
}

// Identify 声明当前节点的身份和版本
// role 当前节点的角色
// version 编译的版本，建议每次发布时更新版本
func Identify(role string, versions ...string) {
	infra.identify(role, versions...)
}

// Configure 开放修改默认配置
// 比如，在代码中就可以设置一些默认配置
// 这样就可以最大化的减少配置文件的依赖
func Configure(cfg Map) {
	infra.configure(cfg)
}

func Setting() Map {
	return infra.setting()
}

// Ready 准备好各模块
// 当你需要写一个临时程序，但是又需要使用程序里的代码
// 比如，导入老数据，整理文件或是数据，临时的采集程序等等
// 就可以在临时代码中，调用infra.Ready()，然后做你需要做的事情
func Ready() {
	infra.parse()
	// infra.cluster()
	infra.initialize()
	infra.connect()
}

// Go 直接开跑
func Go(args ...string) {
	if l := len(args); l > 0 {
		if l == 1 {
			//role
			infra.identify(args[0])
		} else {
			//role, version
			infra.identify(args[0], args[1])
		}
	}

	infra.parse()
	// infra.cluster()
	infra.initialize()
	infra.connect()
	infra.launch()
	infra.waiting()
	infra.terminate()
}

func Name() string {
	return infra.config.name
}
func Role() string {
	return infra.config.role
}
func Node() string {
	return infra.config.node
}
func Version() string {
	return infra.config.version
}
func Secret() string {
	return infra.config.secret
}
func Salt() string {
	return infra.config.salt
}
func Mode() env {
	return infra.config.mode
}
func Developing() bool {
	return infra.config.mode == developing
}
func Preview() bool {
	return infra.config.mode == preview
}
func Testing() bool {
	return infra.config.mode == testing
}
func Production() bool {
	return infra.config.mode == production
}
