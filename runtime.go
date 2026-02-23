package bamgoo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"

	. "github.com/bamgoo/base"
)

// bamgoo is the bamgoo runtime instance that drives module lifecycle.
var bamgoo = &bamgooRuntime{
	modules: make([]Module, 0),
	project: BAMGOO, profile: GLOBAL, node: "", setting: Map{},
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
		Project string `json:"project"`
		Profile string `json:"profile"`
		Node    string `json:"node"`
	}
)

type bamgooRuntime struct {
	mutex   sync.RWMutex
	modules []Module

	project    string
	profile    string
	profileSet bool
	node       string
	nodeSet    bool
	setting    Map

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
	return c.project
}

func (c *bamgooRuntime) Project() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.project
}

func (c *bamgooRuntime) Profile() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.profile
}

func (c *bamgooRuntime) setProfile(profile string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if profile == "" {
		return
	}
	c.profile = profile
	c.profileSet = true
}

func (c *bamgooRuntime) setNode(node string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if node == "" {
		return
	}
	c.node = node
	c.nodeSet = true
}

func (c *bamgooRuntime) Node() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.node
}

func (c *bamgooRuntime) Identity() bamgooIdentity {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return bamgooIdentity{
		Project: c.project,
		Profile: c.profile,
		Node:    c.node,
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

	if project, ok := cfg["project"].(string); ok && project != "" {
		c.project = project
	}
	if name, ok := cfg["name"].(string); ok && name != "" {
		c.project = name
	}
	if node, ok := cfg["node"].(string); ok && node != "" && !c.nodeSet {
		c.node = node
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

	// bootstrap runtime node from CLI/env before config load, so config can't override it.
	if node, ok := bootstrapNode(); ok {
		c.setNode(node)
	}

	//从配置模块加载配置
	cfg, err := hook.LoadConfig()
	if err != nil {
		panic(fmt.Errorf("load config failed: %w", err))
	}
	c.Config(cfg)

	// ensure node is always available and concise by default.
	c.mutex.Lock()
	if strings.TrimSpace(c.node) == "" {
		c.node = defaultNodeID()
	}
	c.mutex.Unlock()

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

	project, profile, node := c.runtimeInfo()
	fmt.Printf("bamgoo started: project=%s profile=%s node=%s\n", project, profile, node)

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
	project, profile, node := c.runtimeInfo()
	// close the modules in reverse order
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Close()
	}
	c.closeStatus = true
	c.openStatus = false
	c.setupStatus = false
	fmt.Printf("bamgoo stopped: project=%s profile=%s node=%s\n", project, profile, node)
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

func (c *bamgooRuntime) runtimeInfo() (string, string, string) {
	c.mutex.RLock()
	project := c.project
	profile := c.profile
	node := c.node
	c.mutex.RUnlock()
	if project == "" {
		project = BAMGOO
	}
	if profile == "" {
		profile = GLOBAL
	}
	if node == "" {
		node = "-"
	}
	return project, profile, node
}

func bootstrapNode() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("BAMGOO_NODE")); v != "" {
		return v, true
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		kv := strings.TrimPrefix(arg, "--")
		if strings.HasPrefix(kv, "node=") {
			v := strings.TrimSpace(strings.TrimPrefix(kv, "node="))
			if v != "" {
				return v, true
			}
			continue
		}
		if kv == "node" && i+1 < len(args) {
			v := strings.TrimSpace(args[i+1])
			if v != "" && !strings.HasPrefix(v, "--") {
				return v, true
			}
		}
	}
	return "", false
}

func defaultNodeID() string {
	// 48-bit random hex id, always 12 chars and avoid leading zero.
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err == nil {
		if buf[0] == 0 {
			buf[0] = 1
		}
		return hex.EncodeToString(buf)
	}

	// fallback (should be rare): still avoid leading zero.
	id := strings.TrimSpace(Generate())
	if id == "" {
		return "100000000000"
	}
	if len(id) >= 12 {
		id = id[len(id)-12:]
	}
	id = strings.TrimLeft(id, "0")
	if id == "" {
		id = "1"
	}
	if len(id) > 12 {
		id = id[len(id)-12:]
	}
	return strings.Repeat("1", 12-len(id)) + id
}
