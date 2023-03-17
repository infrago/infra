package infra

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	. "github.com/infrago/base"
	"github.com/infrago/util"
)

var (
	infra = &kernel{
		config: kernelConfig{
			name: INFRA, role: INFRA, node: "", version: "",
			secret: INFRA, salt: INFRA, setting: Map{},
		},
		runtime: kernelRuntime{override: true},
		modules: make([]infraModule, 0),
	}
)

type (
	kernel struct {
		// mutex 读写锁
		mutex sync.RWMutex
		// config infra配置
		config kernelConfig
		// runtime 运行时
		// 记录运行状态
		runtime kernelRuntime
		// modules 模块
		// 记录加载的模块列表
		modules []infraModule
	}
	kernelRuntime struct {
		override bool

		// parsed
		// 是否解析过了
		parsed bool

		// initialized
		// 是否初始化
		initialized bool

		// connected
		// 是否已连接
		connected bool

		// launched
		// 是否已运行
		launched bool
	}
	kernelConfig struct {
		// name 项目名称
		name string

		// role 节点角色
		role string

		// node 节点ID，全局唯一
		node string

		// secret 密钥，有些时候需要
		secret string

		// salt 加盐，有些加密的方法需要
		salt string

		//mode 运行模式
		mode env

		// version 节点版本
		version string

		// config 节点配置key
		// 一般是为了远程获取配置
		config string

		// setting 设置，主要是自定义的setting
		// 实际业务代码中一般需要用的配置
		setting Map
	}
	infraModule interface {
		// Register 注册
		// 注册模块或组件
		Register(name string, value Any)

		// Configure 配置
		Configure(Map)

		// Initialize 初始化
		Initialize()

		// Connect 连接驱动
		Connect()

		// Launch 启动
		Launch()

		// Terminate 终止
		Terminate()
	}
)

// 终止顺序需要和初始化顺序相反以保证各模块依赖
func (this *kernel) override(args ...bool) bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if len(args) > 0 {
		this.runtime.override = args[0]
	}
	return this.runtime.override
}

// setting 获取setting
func (this *kernel) setting() Map {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	// 深度复制setting
	return util.DeepMapping(this.config.setting)
}

// loading 装载模块
// 遍历所有已经注册过的模块，避免重复注册
func (this *kernel) loading(mod infraModule) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	//判断是否装载过此模块
	for _, m := range this.modules {
		if m == mod {
			return
		}
	}

	this.modules = append(this.modules, mod)
}

// register 遍历所有模块调用注册
// 动态参数，以支持以下几种可能性
// 并且此方法兼容configure，为各模块加载默认配置
// (string,any) 包括name的注册
// (any)    不包括name的注册
// (anys)    批量注册多个
func (this *kernel) register(args ...Any) {
	name := ""
	loads := make([]Any, 0)

	for _, arg := range args {
		switch vvv := arg.(type) {
		case string:
			name = vvv
		default:
			loads = append(loads, vvv)
		}
	}

	for _, load := range loads {
		if mmm, ok := load.(Map); ok {
			// 兼容所有模块的配置注册
			this.configure(mmm)
		} else if mod, ok := load.(infraModule); ok {
			// 兼容所有模块的配置注册
			this.loading(mod)
		} else {
			//下发到各个模块中注册
			for _, mod := range this.modules {
				mod.Register(name, load)
			}
		}
	}

}

// parse 解析启动参数，参数有以下几个来源
// 1 命令行参数，直接读取启动参数
// 2 环境变量，各种参数单独传过来
// 3 环境变量，就像命令行参数一样，整个传过来
// 主要是方便在docker中启动，或是其它容器
// 以上功能待处理
func (this *kernel) parse() {
	if this.runtime.parsed {
		return
	}

	//初始空配置
	this.configure()

	args := os.Args

	// 1读环境变量
	//

	// 2读命令行参数
	// // 参数大于2个，就解析参数
	// if len(args) > 2 {
	//     var name string
	//     var node string
	//     var bind string
	//     var key string
	//     var tags []string
	//     var join []string

	//     flag.StringVar(&name, "name", "infra", "cluster name")
	//     flag.StringVar(&node, "node", "test", "node name")
	//     flag.StringVar(&bind, "bind", "0.0.0.0:3000", "address to bind listeners to")
	//     flag.StringVar(&key, "key", "", "encryption key")
	//     flag.Var((*flagSlice)(&tags), "tag", "tag pair, specified as key=value")
	//     flag.Var((*flagSlice)(&join), "join", "address of agent to join on startup")

	//     flag.Parse()

	// }

	// 3读配置文件
	// 定义一个文件列表，尝试读取配置
	files := []string{
		"config.toml", "infra.toml", "infrago.toml",
		"config.conf", "infra.conf", "infrago.conf",
	}
	// 如果参数个数为1，表示没有传参数，使用文件名
	if len(args) == 1 {
		base := util.NameWithoutExt(args[0])
		files = append([]string{
			base + ".toml", base + ".conf",
		}, files...)
	}
	// 如果参数个数为2，就是指定了配置文件
	if len(args) >= 2 {
		files = append([]string{args[1]}, files...)
	}

	// 遍历文件
	for _, file := range files {
		// 判断文件是否存在
		if _, err := os.Stat(file); err == nil {
			// 读取文件
			bytes, err := ioutil.ReadFile(file)
			if err == nil {
				// 加载配置，并中断循环，只读取第一个读到的文件
				config, err := util.ParseTOML(string(bytes))
				if err == nil {
					this.configure(config)
					break
				}
			}
		}
	}

}

// identify 声明当前节点的角色和版本
// role 当前节点的角色
// version 编译的版本，建议每次发布时更新版本
// 通常在一个项目中会有多个不同模块（角色），每个模块可能会运行N个节点
// 在集群中标明当前节点的角色和版本，方便管理集群
func (this *kernel) identify(role string, versions ...string) {
	this.config.role = role
	if len(versions) > 0 {
		this.config.version = versions[0]
	}
}

// configure 为所有模块加载配置
// 此方法有可能会被多次调用，解析文件后可调用
// 从配置中心获取到配置后，也会调用
func (this *kernel) configure(args ...Map) {
	config := Map{}
	if len(args) > 0 {
		config = args[0]
	}

	// 如果已经初始化就不让修改了
	if this.runtime.initialized || this.runtime.launched {
		return
	}

	//项目名称
	if name, ok := config["name"].(string); ok {
		if name != this.config.name {
			this.config.name = name
			this.config.secret = name
		}
	}

	//节点ID，可以自己指定
	if node, ok := config["node"].(string); ok && node != "" {
		this.config.node = node
	}
	if secret, ok := config["secret"].(string); ok && secret != "" {
		this.config.secret = secret
	}
	if salt, ok := config["salt"].(string); ok && salt != "" {
		this.config.salt = salt
	}
	if mode, ok := config["mode"].(string); ok {
		mode = strings.ToLower(mode)
		if mode == "t" || mode == "test" || mode == "testing" {
			this.config.mode = testing
		} else if mode == "pre" || mode == "preview" {
			this.config.mode = preview
		} else if mode == "pro" || mode == "prod" || mode == "product" || mode == "production" {
			this.config.mode = production
		} else {
			this.config.mode = developing
		}
	}

	// 配置写到配置中
	if setting, ok := config["setting"].(Map); ok {
		util.DeepMapping(setting, this.config.setting)
	}

	//默认配置
	if this.config.node == "" {
		s := util.SHA1(this.config.name + this.config.role + this.config.version + this.config.secret + this.config.salt)
		this.config.node = s[0:8]
	}

	// 把配置下发到各个模块
	for _, mod := range this.modules {
		mod.Configure(config)
	}
}

// initialize 初始化所有模块
func (this *kernel) initialize() {
	if this.runtime.initialized {
		return
	}
	for _, mod := range this.modules {
		mod.Initialize()
	}
	this.runtime.initialized = true
}

// connect
func (this *kernel) connect() {
	if this.runtime.connected {
		return
	}
	for _, mod := range this.modules {
		mod.Connect()
	}
	this.runtime.connected = true
}

// launch 启动所有模块
// 只有部分模块是需要启动的，比如HTTP
func (this *kernel) launch() {
	if this.runtime.launched {
		return
	}
	for _, mod := range this.modules {
		mod.Launch()
	}

	//这里是触发器
	//待处理，而且异步要走携程池
	go infraTrigger.Toggle(START)

	this.runtime.launched = true

	if this.config.name == this.config.role || this.config.role == "" {
		log.Println(fmt.Sprintf("%s %s-%s is running", INFRAGO, this.config.name, this.config.node))
	} else {
		log.Println(fmt.Sprintf("%s %s-%s-%s is running", INFRAGO, this.config.name, this.config.role, this.config.node))
	}
}

// waiting 等待系统退出信号
// 为了程序做好退出前的善后工作，优雅的退出程序
func (this *kernel) waiting() {
	// 待处理，加入自己的退出信号
	// 并开放 infra.Stop() 给外部调用
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// terminate 终止结束所有模块
// 终止顺序需要和初始化顺序相反以保证各模块依赖
func (this *kernel) terminate() {

	// 停止前触发器，同步
	// 待处理 触发器
	infraTrigger.SyncToggle(STOP)

	//反向停止模块
	for i := len(this.modules) - 1; i >= 0; i-- {
		mod := this.modules[i]
		mod.Terminate()
	}
	this.runtime.launched = false

	if this.config.name == this.config.role || this.config.role == "" {
		log.Println(fmt.Sprintf("%s %s-%s is stopped", INFRAGO, this.config.name, this.config.node))
	} else {
		log.Println(fmt.Sprintf("%s %s-%s-%s is stopped", INFRAGO, this.config.name, this.config.role, this.config.node))
	}
}
