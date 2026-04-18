package util

func GetOrCreateMap(target map[string]interface{}, key string) map[string]interface{} {
	if target == nil {
		return nil
	}
	val, exists := target[key]
	if !exists {
		nested := make(map[string]interface{})
		target[key] = nested
		return nested
	}
	nested, ok := val.(map[string]interface{})
	if !ok {
		// Value exists but is not a map - replace it
		nested = make(map[string]interface{})
		target[key] = nested
	}
	return nested
}

func MergeMap(target, source map[string]interface{}, key string) {
	if len(source) == 0 {
		return
	}

	if key == "" {
		for k, v := range source {
			target[k] = v
		}
		return
	}

	nested := GetOrCreateMap(target, key)
	for k, v := range source {
		nested[k] = v
	}
}
