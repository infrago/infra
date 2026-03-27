package infra

import (
	"strings"
	"sync"

	. "github.com/infrago/base"
)

var upgrade = &upgradeModule{
	acceptors: make(map[string]UpgradeAcceptor),
}

type (
	UpgradeAcceptOptions struct {
		Socket     Any
		Meta       *Meta
		Name       string
		Site       string
		Host       string
		Domain     string
		RootDomain string
		Path       string
		Uri        string
		Setting    Map
		Params     Map
		Query      Map
		Form       Map
		Value      Map
		Args       Map
		Locals     Map
	}

	UpgradeAcceptor func(UpgradeAcceptOptions) error

	upgradeModule struct {
		mutex     sync.RWMutex
		acceptors map[string]UpgradeAcceptor
	}
)

func RegisterUpgradeAcceptor(name string, accept UpgradeAcceptor) {
	if accept == nil {
		return
	}
	name = normalizeUpgradeName(name)

	upgrade.mutex.Lock()
	defer upgrade.mutex.Unlock()

	if infrago.Override() {
		upgrade.acceptors[name] = accept
		return
	}
	if _, ok := upgrade.acceptors[name]; !ok {
		upgrade.acceptors[name] = accept
	}
}

func LoadUpgradeAcceptor(name string) (UpgradeAcceptor, bool) {
	name = normalizeUpgradeName(name)

	upgrade.mutex.RLock()
	defer upgrade.mutex.RUnlock()

	accept, ok := upgrade.acceptors[name]
	return accept, ok
}

func normalizeUpgradeName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return DEFAULT
	}
	return name
}
