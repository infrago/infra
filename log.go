package infra

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	base "github.com/infrago/base"
)

const (
	LogLevelDebug   = "debug"
	LogLevelTrace   = "trace"
	LogLevelInfo    = "info"
	LogLevelNotice  = "notice"
	LogLevelWarning = "warning"
	LogLevelError   = "error"
	LogLevelPanic   = "panic"
	LogLevelFatal   = "fatal"
)

// LogEntry is the framework-wide structured log contract.
type LogEntry struct {
	Time      time.Time `json:"time"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Module    string    `json:"module"`
	Project   string    `json:"project"`
	Role      string    `json:"role"`
	Profile   string    `json:"profile"`
	Node      string    `json:"node"`
	RequestID string    `json:"request_id"`
	TraceID   string    `json:"trace_id"`
	Fields    base.Map  `json:"fields,omitempty"`
}

// LogHook allows a logging module to consume framework log entries.
type LogHook interface {
	Log(LogEntry)
}

type defaultLogHook struct {
	mutex  sync.Mutex
	stdout io.Writer
	stderr io.Writer
}

func newDefaultLogHook() *defaultLogHook {
	return &defaultLogHook{stdout: os.Stdout, stderr: os.Stderr}
}

func (h *defaultLogHook) Log(entry LogEntry) {
	entry = normalizeLogEntry(entry)
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	writer := h.stdout
	switch entry.Level {
	case LogLevelError, LogLevelPanic, LogLevelFatal:
		writer = h.stderr
	}
	if writer == nil {
		return
	}
	h.mutex.Lock()
	_, _ = writer.Write(append(data, '\n'))
	h.mutex.Unlock()
}

// Log emits a structured log without request metadata.
func Log(level, module, message string, fields ...base.Map) {
	LogWith(nil, level, module, message, fields...)
}

// LogWith emits a structured log and enriches it with request and trace metadata.
func LogWith(meta *Meta, level, module, message string, fields ...base.Map) {
	entry := LogEntry{
		Time:    time.Now(),
		Level:   normalizeLogLevel(level),
		Message: message,
		Module:  normalizeLogModule(module),
		Fields:  mergeLogFields(fields...),
	}
	if meta != nil {
		entry.RequestID = meta.RequestId()
		entry.TraceID = meta.TraceId()
	}
	hook.Log(normalizeLogEntry(entry))
}

// Log emits a structured log associated with this request metadata.
func (m *Meta) Log(level, module, message string, fields ...base.Map) {
	LogWith(m, level, module, message, fields...)
}

func normalizeLogEntry(entry LogEntry) LogEntry {
	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}
	entry.Level = normalizeLogLevel(entry.Level)
	entry.Module = normalizeLogModule(entry.Module)
	entry.Fields = mergeLogFields(entry.Fields)
	identity := Identity()
	if strings.TrimSpace(entry.Project) == "" {
		entry.Project = identity.Project
	}
	if strings.TrimSpace(entry.Role) == "" {
		entry.Role = identity.Role
	}
	if strings.TrimSpace(entry.Profile) == "" {
		entry.Profile = identity.Profile
	}
	if strings.TrimSpace(entry.Node) == "" {
		entry.Node = identity.Node
	}
	return entry
}

func normalizeLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case LogLevelDebug:
		return LogLevelDebug
	case LogLevelTrace:
		return LogLevelTrace
	case LogLevelNotice:
		return LogLevelNotice
	case LogLevelWarning, "warn":
		return LogLevelWarning
	case LogLevelError:
		return LogLevelError
	case LogLevelPanic:
		return LogLevelPanic
	case LogLevelFatal:
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

func normalizeLogModule(module string) string {
	module = strings.ToLower(strings.TrimSpace(module))
	if module == "" {
		return "application"
	}
	return module
}

func mergeLogFields(items ...base.Map) base.Map {
	out := base.Map{}
	for _, fields := range items {
		for key, value := range fields {
			out[key] = value
		}
	}
	return out
}
