package infra

import (
	"errors"
	"time"

	. "github.com/infrago/base"
)

var (
	ErrUnavaliableBridge = errors.New("Unavaliable bridge.")
)

var (
	infraBridge = &bridgeModule{}
)

type (
	bridgeModule struct {
		bus   BusBridge
		token tokenBridge
		log   logBridge
	}

	tokenBridge interface {
		Validate(token string) error
	}
	BusBridge interface {
		Request(meta Metadata, timeout time.Duration) (*Echo, error)
	}

	logBridge interface {
		Console(args ...Any)
		Debug(args ...Any)
		Trace(args ...Any)
		Info(args ...Any)
		Notice(args ...Any)
		Warning(args ...Any)
		Error(args ...Any)
		Panic(args ...Any)
		Fatal(args ...Any)
	}
)

// Register
func (this *bridgeModule) Register(o Object) {
	switch val := o.Object.(type) {
	case BusBridge:
		this.bus = val
	case logBridge:
		this.log = val
	case tokenBridge:
		this.token = val
	}
}
func (this *bridgeModule) Configure(config Map) {
}
func (this *bridgeModule) Initialize() {
}
func (this *bridgeModule) Connect() {
}
func (this *bridgeModule) Launch() {
}
func (this *bridgeModule) Terminate() {
}

func (this *bridgeModule) Request(meta *Meta, name string, value Map, timeout time.Duration) (*Echo, error) {
	if this.bus == nil {
		return nil, ErrUnavaliableBridge
	}

	data := meta.Metadata()
	data.Name = name
	data.Payload = value

	return this.bus.Request(data, timeout)
}

//--------------- log bridge begin ---------------------------------------------------

func (this *bridgeModule) Console(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Console(args...)
}
func (this *bridgeModule) Debug(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Debug(args...)
}
func (this *bridgeModule) Trace(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Trace(args...)
}
func (this *bridgeModule) Info(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Info(args...)
}
func (this *bridgeModule) Notice(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Notice(args...)
}
func (this *bridgeModule) Warning(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Warning(args...)
}
func (this *bridgeModule) Error(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Error(args...)
}

func (this *bridgeModule) Panic(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Panic(args...)
}
func (this *bridgeModule) Fatal(args ...Any) {
	if this.log == nil {
		return
	}
	this.log.Fatal(args...)
}

//-----------------------------------------------------

func Console(args ...Any) {
	infraBridge.Console(args...)
}
func Debug(args ...Any) {
	infraBridge.Debug(args...)
}
func Trace(args ...Any) {
	infraBridge.Trace(args...)
}
func Info(args ...Any) {
	infraBridge.Info(args...)
}
func Notice(args ...Any) {
	infraBridge.Notice(args...)
}
func Warning(args ...Any) {
	infraBridge.Warning(args...)
}
func Error(args ...Any) {
	infraBridge.Error(args...)
}

func Panic(args ...Any) {
	infraBridge.Panic(args...)
}
func Fatal(args ...Any) {
	infraBridge.Fatal(args...)
}

//--------------- log bridge end ---------------------------------------------------
