package infra

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	base "github.com/infrago/base"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type defaultBusHook struct{}

type defaultConfigHook struct{}
type defaultTraceHook struct{}

func (h *defaultBusHook) Request(meta *Meta, name string, value base.Map, _ time.Duration) (base.Map, base.Res) {
	data, res, ok := core.invokeLocalWithKinds(meta, name, value, []string{coreKindService})
	if ok {
		return data, res
	}
	return nil, OK
}

func (h *defaultBusHook) Broadcast(meta *Meta, name string, value base.Map) error {
	_, _, _ = core.invokeLocalWithKinds(meta, name, value, []string{coreKindMessage})
	return nil
}

func (h *defaultBusHook) Rolecast(meta *Meta, name string, value base.Map) error {
	_, _, _ = core.invokeLocalWithKinds(meta, name, value, []string{coreKindMessage})
	return nil
}

func (h *defaultBusHook) Dispatch(meta *Meta, name string, value base.Map) error {
	retries := core.dispatchRetries(name)
	go h.dispatchService(meta, name, value, retries, 1)
	return nil
}

func (h *defaultBusHook) Publish(meta *Meta, name string, value base.Map) error {
	return h.Rolecast(meta, name, value)
}

func (h *defaultBusHook) Enqueue(meta *Meta, name string, value base.Map) error {
	return h.Dispatch(meta, name, value)
}

func (h *defaultBusHook) dispatchService(meta *Meta, name string, value base.Map, retries []time.Duration, attempt int) {
	if attempt <= 0 {
		attempt = 1
	}

	localMeta := NewMeta()
	if meta != nil {
		localMeta.Metadata(meta.Metadata())
	}

	setting := base.Map{
		dispatchAttemptSetting: attempt,
		dispatchFinalSetting:   dispatchFinal(retries, attempt),
	}
	_, res, found := core.invokeLocalWithKinds(localMeta, name, value, []string{coreKindService}, setting)
	if !found || !dispatchRetryableResult(res) {
		return
	}

	delay, ok := dispatchRetryDelay(retries, attempt)
	if !ok {
		return
	}
	time.AfterFunc(delay, func() {
		h.dispatchService(meta, name, value, retries, attempt+1)
	})
}

func (h *defaultBusHook) Stats() []ServiceStats {
	return nil
}

func (h *defaultBusHook) ListNodes() []NodeInfo {
	return nil
}

func (h *defaultBusHook) ListServices() []ServiceInfo {
	return nil
}

func (h *defaultConfigHook) LoadConfig() (base.Map, error) {
	drvName, params, err := parseConfigParams()
	if err != nil {
		return nil, err
	}
	if drvName == "" {
		return nil, nil
	}
	if drvName != DEFAULT && drvName != "file" {
		return nil, errors.New("Unknown config driver: " + drvName)
	}
	return loadConfigFromFile(params)
}

func dispatchRetryableResult(res base.Res) bool {
	if IsRetry(res) {
		return true
	}
	if res == nil || !res.Fail() {
		return false
	}
	switch res.Status() {
	case Invalid.Status(), Denied.Status(), Unsigned.Status(), Unauthed.Status():
		return false
	}
	return true
}

func dispatchFinal(retries []time.Duration, attempt int) bool {
	if len(retries) == 0 {
		return false
	}
	if attempt <= 0 {
		attempt = 1
	}
	return attempt > len(retries)
}

func dispatchRetryDelay(retries []time.Duration, attempt int) (time.Duration, bool) {
	if len(retries) == 0 {
		return 0, false
	}
	if attempt <= 0 {
		attempt = 1
	}
	idx := attempt - 1
	if idx >= len(retries) {
		return 0, false
	}
	delay := retries[idx]
	if delay <= 0 {
		delay = time.Second
	}
	return delay, true
}

func (h *defaultTraceHook) Begin(_ *Meta, _ string, _ base.Map) TraceSpan {
	return noopTraceSpan{}
}

func (h *defaultTraceHook) Trace(_ *Meta, _ string, _ string, _ base.Map) error {
	return nil
}

func parseConfigParams() (string, base.Map, error) {
	params := base.Map{}
	for k, v := range parseConfigEnv() {
		params[k] = v
	}
	for k, v := range parseConfigArgs() {
		params[k] = v
	}

	driver := DEFAULT
	if v, ok := params["driver"].(string); ok && v != "" {
		driver = v
	}
	if driver == "" {
		driver = DEFAULT
	}
	if driver == DEFAULT || driver == "file" {
		if _, ok := params["file"]; !ok {
			if file := defaultConfigFile(); file != "" {
				params["file"] = file
			}
		}
	}
	return driver, params, nil
}

func parseConfigEnv() base.Map {
	envs := os.Environ()
	params := base.Map{}
	for _, kv := range envs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		if !strings.HasPrefix(key, "INFRAGO_") {
			continue
		}
		if strings.TrimSpace(val) == "" {
			continue
		}
		k := normalizeConfigParamKey(strings.TrimPrefix(key, "INFRAGO_"))
		params[k] = val
	}
	return params
}

func parseConfigArgs() base.Map {
	args := os.Args[1:]
	params := base.Map{}

	if len(args) == 1 {
		if isConfigFileArg(args[0]) {
			params["driver"] = DEFAULT
			params["file"] = args[0]
			return params
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			if i == 0 {
				params["driver"] = arg
			}
			continue
		}
		kv := strings.TrimPrefix(arg, "--")
		if kv == "" {
			continue
		}
		if strings.Contains(kv, "=") {
			parts := strings.SplitN(kv, "=", 2)
			params[normalizeConfigParamKey(parts[0])] = parts[1]
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			params[normalizeConfigParamKey(kv)] = args[i+1]
			i++
		} else {
			params[normalizeConfigParamKey(kv)] = "true"
		}
	}
	return params
}

func normalizeConfigParamKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, ".", "_")
	switch key {
	case "config_driver":
		return "driver"
	case "config_file", "configfile":
		return "file"
	case "config_path", "configpath":
		return "path"
	case "config_addr", "redis_addr", "redisaddr":
		return "addr"
	case "redis_host":
		return "host"
	case "redis_server":
		return "server"
	case "redis_port":
		return "port"
	}
	return key
}

func isConfigFileArg(arg string) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" || strings.HasPrefix(arg, "-") {
		return false
	}
	if _, err := os.Stat(arg); err == nil {
		return true
	}
	switch strings.ToLower(filepath.Ext(arg)) {
	case ".json", ".toml", ".tml", ".yaml", ".yml":
		return true
	}
	return strings.ContainsAny(arg, `/\`)
}

func loadConfigFromFile(params base.Map) (base.Map, error) {
	file := ""
	if vv, ok := params["file"].(string); ok {
		file = vv
	}
	if vv, ok := params["path"].(string); ok {
		file = vv
	}
	if vv, ok := params["config"].(string); ok {
		file = vv
	}
	if file == "" {
		file = defaultConfigFile()
	}
	if file == "" {
		return nil, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", file, err)
	}
	format, _ := params["format"].(string)
	if format == "" {
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".json":
			format = "json"
		case ".toml", ".tml":
			format = "toml"
		case ".yaml", ".yml":
			format = "yaml"
		}
	}
	if format == "" {
		format = detectConfigFormat(data)
	}
	return decodeConfig(data, format)
}

func defaultConfigFile() string {
	candidates := []string{"config.toml", "config.json", "config.yaml", "config.yml"}
	if exe := filepath.Base(os.Args[0]); exe != "" {
		name := strings.TrimSuffix(exe, filepath.Ext(exe))
		candidates = append(candidates, name+".toml", name+".json", name+".yaml", name+".yml")
	}
	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	return ""
}

func detectConfigFormat(data []byte) string {
	str := strings.TrimSpace(string(data))
	if str == "" {
		return ""
	}
	if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
		return "json"
	}
	if looksLikeToml(str) {
		return "toml"
	}
	if looksLikeYaml(str) {
		return "yaml"
	}
	if str != "" {
		return "toml"
	}
	return ""
}

func decodeConfig(data []byte, format string) (base.Map, error) {
	if strings.TrimSpace(string(data)) == "" {
		return base.Map{}, nil
	}

	format = strings.ToLower(strings.TrimSpace(format))
	var out base.Map
	switch format {
	case "json":
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "toml":
		if err := toml.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, errors.New("Unknown config format: " + format)
	}
}

var (
	tomlKeyValPattern = regexp.MustCompile(`(?m)^\s*[\w\.\-]+\s*=`)
	tomlSection       = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]\s*$`)
	yamlKeyValPattern = regexp.MustCompile(`(?m)^\s*[\w\.\-]+\s*:\s*`)
	yamlListPattern   = regexp.MustCompile(`(?m)^\s*-\s+`)
)

func looksLikeToml(s string) bool {
	return tomlKeyValPattern.MatchString(s) || tomlSection.MatchString(s)
}

func looksLikeYaml(s string) bool {
	return yamlKeyValPattern.MatchString(s) || yamlListPattern.MatchString(s)
}
