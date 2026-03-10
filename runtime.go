package infra

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"

	. "github.com/infrago/base"
)

// infrago is the infrago runtime instance that drives module lifecycle.
var infrago = &infragoRuntime{
	modules: make([]Module, 0),
	project: INFRAGO, profile: GLOBAL, role: GLOBAL, node: "", setting: Map{}, runProfiles: []string{GLOBAL},
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

	infragoIdentity struct {
		Project string `json:"project"`
		Role    string `json:"role"`
		Profile string `json:"profile"`
		Node    string `json:"node"`
	}
)

type infragoRuntime struct {
	mutex   sync.RWMutex
	modules []Module

	project       string
	role          string
	profile       string
	runProfiles   []string
	configRole    string
	configProfile string
	node          string
	nodeSet       bool
	setting       Map

	overrideStatus bool
	loadStatus     bool
	configStatus   bool
	setupStatus    bool
	openStatus     bool
	startStatus    bool
	closeStatus    bool
}

func (c *infragoRuntime) Name() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.project
}

func (c *infragoRuntime) Project() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.project
}

func (c *infragoRuntime) Profile() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.profile
}

func (c *infragoRuntime) Role() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.role
}

func (c *infragoRuntime) setRequestedProfiles(profiles []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(profiles) == 0 {
		c.runProfiles = []string{GLOBAL}
		return
	}
	next := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		profile = normalizeToken(profile)
		if profile == "" {
			continue
		}
		next = append(next, profile)
	}
	if len(next) == 0 {
		next = []string{GLOBAL}
	}
	c.runProfiles = next
}

func (c *infragoRuntime) setNode(node string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if node == "" {
		return
	}
	c.node = node
	c.nodeSet = true
}

func (c *infragoRuntime) Node() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.node
}

func (c *infragoRuntime) Identity() infragoIdentity {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return infragoIdentity{
		Project: c.project,
		Role:    c.role,
		Profile: c.profile,
		Node:    c.node,
	}
}

// Mount attaches a module into the core lifecycle and returns a host for submodules.
func (c *infragoRuntime) Mount(mod Module) Host {
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
func (c *infragoRuntime) Register(name string, value Any) {
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
func (c *infragoRuntime) runtimeConfig(cfg Map) {
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
		mergeMap(c.setting, setting)
	}
	if profile, ok := cfg["profile"].(string); ok {
		profile = normalizeToken(profile)
		if profile != "" {
			c.configProfile = profile
		}
	}
	if role, ok := cfg["role"].(string); ok {
		role = normalizeToken(role)
		if role != "" {
			c.configRole = role
		}
	}
	if runtimeCfg, ok := cfg["infrago"].(Map); ok {
		if profile, ok := runtimeCfg["profile"].(string); ok {
			profile = normalizeToken(profile)
			if profile != "" {
				c.configProfile = profile
			}
		}
		if role, ok := runtimeCfg["role"].(string); ok {
			role = normalizeToken(role)
			if role != "" {
				c.configRole = role
			}
		}
	}

	c.configStatus = true
}

// Load 加载配置
func (c *infragoRuntime) Load() {
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
	if selected := c.effectiveProfilesLocked(); len(selected) > 0 {
		c.profile = selected[0]
	}
	c.role = c.effectiveRoleLocked(c.profile)
	c.mutex.Unlock()

	c.loadStatus = true
}

// Config applies config to core and all modules.
func (c *infragoRuntime) Config(cfg Map) {
	if cfg == nil {
		cfg = Map{}
	}

	c.runtimeConfig(cfg)
	for _, mod := range c.modules {
		mod.Config(cfg)
	}
}

// Setup initializes all modules.
func (c *infragoRuntime) Setup() {
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
func (c *infragoRuntime) Open() {
	if c.openStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Open()
	}
	c.openStatus = true
}

// Start launches all modules.
func (c *infragoRuntime) Start() {
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

	project, role, profile, node := c.runtimeInfo()
	fmt.Printf("infrago started: project=%s role=%s profile=%s node=%s\n", project, role, profile, node)

	c.startStatus = true
}

// Stop terminates all modules in reverse order.
func (c *infragoRuntime) Stop() {
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
func (c *infragoRuntime) Close() {
	if c.closeStatus {
		return
	}
	project, role, profile, node := c.runtimeInfo()
	// close the modules in reverse order
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Close()
	}
	c.closeStatus = true
	c.openStatus = false
	c.setupStatus = false
	fmt.Printf("infrago stopped: project=%s role=%s profile=%s node=%s\n", project, role, profile, node)
}

// Wait blocks until system termination signal.
func (c *infragoRuntime) Wait() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()
	<-ctx.Done()
}

// Override controls whether registrations can overwrite existing entries.
func (c *infragoRuntime) Override(args ...bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(args) > 0 {
		c.overrideStatus = args[0]
	}
	return c.overrideStatus
}

func (c *infragoRuntime) runtimeInfo() (string, string, string, string) {
	c.mutex.RLock()
	project := c.project
	role := c.role
	profile := c.profile
	node := c.node
	c.mutex.RUnlock()
	if project == "" {
		project = INFRAGO
	}
	if profile == "" {
		profile = GLOBAL
	}
	if role == "" {
		role = profile
	}
	if node == "" {
		node = "-"
	}
	return project, role, profile, node
}

func (c *infragoRuntime) EffectiveProfiles() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.effectiveProfilesLocked()
}

func (c *infragoRuntime) effectiveProfilesLocked() []string {
	// priority: env > config > Run(profile) > global
	if profile, ok := bootstrapProfile(); ok {
		return []string{profile}
	}
	if c.configProfile != "" {
		return []string{c.configProfile}
	}
	if len(c.runProfiles) > 0 {
		out := make([]string, len(c.runProfiles))
		copy(out, c.runProfiles)
		return out
	}
	return []string{GLOBAL}
}

func (c *infragoRuntime) effectiveRoleLocked(profile string) string {
	if role, ok := bootstrapRole(); ok {
		return role
	}
	if c.configRole != "" {
		return c.configRole
	}
	if profile != "" {
		return profile
	}
	return GLOBAL
}

func bootstrapNode() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("INFRAGO_NODE")); v != "" {
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

func bootstrapProfile() (string, bool) {
	if v := normalizeToken(os.Getenv("INFRAGO_PROFILE")); v != "" {
		return v, true
	}
	return "", false
}

func bootstrapRole() (string, bool) {
	if v := normalizeToken(os.Getenv("INFRAGO_ROLE")); v != "" {
		return v, true
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		kv := strings.TrimPrefix(arg, "--")
		if strings.HasPrefix(kv, "role=") {
			v := normalizeToken(strings.TrimSpace(strings.TrimPrefix(kv, "role=")))
			if v != "" {
				return v, true
			}
			continue
		}
		if kv == "role" && i+1 < len(args) {
			v := normalizeToken(strings.TrimSpace(args[i+1]))
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
