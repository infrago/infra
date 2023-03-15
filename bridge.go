package infra

import (
	"errors"
	"time"

	. "github.com/infrago/base"
)

var (
	infraBridge = &bridgeModule{}

	errUnavaliableBusBridge = errors.New("Unavaliable bus bridge.")
)

type (
	bridgeModule struct {
		bus   BusBridge
		token tokenBridge
	}

	BusBridge interface {
		Request(meta Metadata, timeout time.Duration) (*Echo, error)
	}

	tokenBridge interface {
		Validate(token string) error
	}
)

// Register
func (this *bridgeModule) Register(name string, value Any) {
	switch val := value.(type) {
	case BusBridge:
		this.bus = val
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
		return nil, errUnavaliableBusBridge
	}

	data := meta.Metadata()
	data.Name = name
	data.Payload = value

	return this.bus.Request(data, timeout)
}
