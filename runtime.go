package bamgoo

import (
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	. "github.com/bamgoo/base"
)

// bamgoo is the bamgoo runtime instance that drives module lifecycle.
var bamgoo = &bamgooRuntime{
	modules: make([]Module, 0),
	name:    BAMGOO, role: BAMGOO, node: "", version: "", setting: Map{},
}

type (
	Module interface {
		Register(string, Any)
		Config(Map)
		Setup()
		Open()
		Start()
		Stop()
		Close()
	}

	bamgooIdentity struct {
		Name    string `json:"name"`
		Role    string `json:"role"`
		Node    string `json:"node"`
		Version string `json:"version"`
	}
)

type bamgooRuntime struct {
	mutex   sync.RWMutex
	modules []Module

	name    string
	role    string
	node    string
	version string
	setting Map

	overrideStatus bool
	loadStatus     bool
	configStatus   bool
	setupStatus    bool
	openStatus     bool
	startStatus    bool
	closeStatus    bool
}

func (c *bamgooRuntime) Name() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.name
}

func (c *bamgooRuntime) Project() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.name
}

func (c *bamgooRuntime) Role() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.role
}

func (c *bamgooRuntime) Node() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.node
}

func (c *bamgooRuntime) Version() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.version
}

func (c *bamgooRuntime) Identity() bamgooIdentity {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return bamgooIdentity{
		Name:    c.name,
		Role:    c.role,
		Node:    c.node,
		Version: c.version,
	}
}

// Mount attaches a module into the core lifecycle and returns a host for submodules.
func (c *bamgooRuntime) Mount(mod Module) Host {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if slices.Contains(c.modules, mod) {
		panic("模块已经挂载了.")
	}

	// if the value is a hook, register it
	hook.Attach(mod)

	// append the module to the modules list
	c.modules = append(c.modules, mod)

	return host
}

// Register dispatches registrations to all mounted modules.
func (c *bamgooRuntime) Register(name string, value Any) {
	// if the value is a module, mount it
	if mod, ok := value.(Module); ok {
		c.Mount(mod)
		return
	}

	// if the value is a config, update the config
	if cfg, ok := value.(Map); ok {
		c.Config(cfg)
	}

	// dispatch the registration to all mounted modules
	for _, mod := range c.modules {
		mod.Register(name, value)
	}
}

// Config updates core config and broadcasts to modules.
func (c *bamgooRuntime) runtimeConfig(cfg Map) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.setupStatus || c.openStatus || c.startStatus {
		return
	}

	if cfg == nil {
		cfg = Map{}
	}

	if name, ok := cfg["name"].(string); ok && name != "" {
		c.name = name
	}
	if role, ok := cfg["role"].(string); ok {
		c.role = role
	}
	if node, ok := cfg["node"].(string); ok && node != "" {
		c.node = node
	}
	if version, ok := cfg["version"].(string); ok {
		c.version = version
	}
	if setting, ok := cfg["setting"].(Map); ok {
		for k, v := range setting {
			c.setting[k] = v
		}
	}

	c.configStatus = true
}

// Load 加载配置
func (c *bamgooRuntime) Load() {
	if c.loadStatus {
		return
	}

	//从配置模块加载配置
	cfg, err := hook.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("load config failed: %w", err))
	}
	c.Config(cfg)

	c.loadStatus = true
}

// Config applies config to core and all modules.
func (c *bamgooRuntime) Config(cfg Map) {
	if cfg == nil {
		cfg = Map{}
	}

	c.runtimeConfig(cfg)
	for _, mod := range c.modules {
		mod.Config(cfg)
	}
}

// Setup initializes all modules.
func (c *bamgooRuntime) Setup() {
	if c.setupStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Setup()
	}
	c.setupStatus = true
	c.closeStatus = false
}

// Open connects all modules.
func (c *bamgooRuntime) Open() {
	if c.openStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Open()
	}
	c.openStatus = true
}

// Start launches all modules.
func (c *bamgooRuntime) Start() {
	if c.startStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Start()
	}
	// Trigger START after all modules are started.
	// This must stay in runtime (not triggerModule.Start), otherwise the
	// trigger can fire before late modules (e.g. bus) are fully ready.
	trigger.Toggle(START)
	c.startStatus = true
}

// Stop terminates all modules in reverse order.
func (c *bamgooRuntime) Stop() {
	if !c.startStatus {
		return
	}
	// Trigger STOP before module shutdown, so handlers can still use modules
	// like bus/log while they are alive.
	// This is centralized here for deterministic lifecycle ordering.
	trigger.SyncToggle(STOP)
	// stop the modules in reverse order
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Stop()
	}
	c.startStatus = false
}

// Close releases resources for all modules in reverse order.
func (c *bamgooRuntime) Close() {
	if c.closeStatus {
		return
	}
	// close the modules in reverse order
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Close()
	}
	c.closeStatus = true
	c.openStatus = false
	c.setupStatus = false
}

// Wait blocks until system termination signal.
func (c *bamgooRuntime) Wait() {
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// Override controls whether registrations can overwrite existing entries.
func (c *bamgooRuntime) Override(args ...bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(args) > 0 {
		c.overrideStatus = args[0]
	}
	return c.overrideStatus
}
