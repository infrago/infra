package infra

import . "github.com/infrago/base"

func (c *infragoRuntime) Setting() Map {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return cloneSettingMap(c.setting)
}

func cloneSettingMap(src Map) Map {
	if src == nil {
		return Map{}
	}

	dst := make(Map, len(src))
	for key, value := range src {
		dst[key] = cloneSettingValue(value)
	}
	return dst
}

func cloneSettingValue(value Any) Any {
	switch v := value.(type) {
	case Map:
		return cloneSettingMap(v)
	case []Map:
		out := make([]Map, len(v))
		for i, item := range v {
			out[i] = cloneSettingMap(item)
		}
		return out
	case []Any:
		out := make([]Any, len(v))
		for i, item := range v {
			out[i] = cloneSettingValue(item)
		}
		return out
	default:
		return value
	}
}

func mergeMap(dst, src Map) Map {
	if dst == nil {
		dst = Map{}
	}
	for key, value := range src {
		if current, ok := dst[key].(Map); ok {
			if next, ok := value.(Map); ok {
				dst[key] = mergeMap(current, next)
				continue
			}
		}
		dst[key] = cloneSettingValue(value)
	}
	return dst
}
