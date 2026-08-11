package infra

import "github.com/infrago/base"

func (m *coreModule) Ready() bool { return true }

func (m *coreModule) Health() ModuleHealth {
	m.mutex.RLock()
	entries := len(m.entries)
	m.mutex.RUnlock()
	return NewModuleHealth("core", true, nil, base.Map{"entries": entries})
}

func (m *coreModule) Stats() ModuleStats {
	m.mutex.RLock()
	entries := len(m.entries)
	m.mutex.RUnlock()
	return NewModuleStats("core", true, base.Map{"entries": entries})
}

func (m *basicModule) Ready() bool { return true }

func (m *basicModule) Health() ModuleHealth {
	m.mutex.Lock()
	statuses := len(m.statuses)
	types := len(m.types)
	m.mutex.Unlock()
	return NewModuleHealth("basic", true, nil, base.Map{"statuses": statuses, "types": types})
}

func (m *basicModule) Stats() ModuleStats {
	m.mutex.Lock()
	statuses := len(m.statuses)
	types := len(m.types)
	m.mutex.Unlock()
	return NewModuleStats("basic", true, base.Map{"statuses": statuses, "types": types})
}

func (m *codecModule) Ready() bool { return true }

func (m *codecModule) Health() ModuleHealth {
	m.mutex.Lock()
	codecs := len(m.codecs)
	m.mutex.Unlock()
	return NewModuleHealth("codec", true, nil, base.Map{"codecs": codecs})
}

func (m *codecModule) Stats() ModuleStats {
	m.mutex.Lock()
	codecs := len(m.codecs)
	m.mutex.Unlock()
	return NewModuleStats("codec", true, base.Map{"codecs": codecs})
}

func (m *libraryModule) Ready() bool { return true }

func (m *libraryModule) Health() ModuleHealth {
	m.mutex.RLock()
	libraries := len(m.libraries)
	m.mutex.RUnlock()
	return NewModuleHealth("library", true, nil, base.Map{"libraries": libraries})
}

func (m *libraryModule) Stats() ModuleStats {
	m.mutex.RLock()
	libraries := len(m.libraries)
	m.mutex.RUnlock()
	return NewModuleStats("library", true, base.Map{"libraries": libraries})
}

func (m *triggerModule) Ready() bool { return true }

func (m *triggerModule) Health() ModuleHealth {
	m.mutex.Lock()
	triggers := len(m.triggers)
	methods := len(m.methods)
	m.mutex.Unlock()
	return NewModuleHealth("trigger", true, nil, base.Map{"triggers": triggers, "methods": methods})
}

func (m *triggerModule) Stats() ModuleStats {
	m.mutex.Lock()
	triggers := len(m.triggers)
	methods := len(m.methods)
	m.mutex.Unlock()
	return NewModuleStats("trigger", true, base.Map{"triggers": triggers, "methods": methods})
}
