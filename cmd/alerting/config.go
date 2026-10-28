package alerting

import (
	"fmt"
	"maps"
	"strings"

	"gopkg.in/yaml.v3"
)

// BuildRenderConfig merges base configuration (such as OpenSLO metadata),
// optional YAML values bytes, --set key=value overrides, and --set-json overrides.
func BuildRenderConfig(base map[string]any, valuesYAML []byte, setVals, setJSONVals []string) (map[string]any, error) {
	configData := make(map[string]any)

	// 1. Copy base config (e.g. from OpenSLO spec)
	maps.Copy(configData, base)

	// 2. Unmarshal values YAML (overrides base defaults)
	if len(valuesYAML) > 0 {
		if err := yaml.Unmarshal(valuesYAML, &configData); err != nil {
			return nil, fmt.Errorf("parsing values file: %w", err)
		}
	}

	// 3. Apply --set flags (overrides file and base defaults)
	if err := ApplySetFlags(configData, setVals); err != nil {
		return nil, err
	}

	// 4. Apply --set-json flags
	if err := ApplySetJSONFlags(configData, setJSONVals); err != nil {
		return nil, err
	}

	return configData, nil
}

// ApplySetFlags parses and applies --set key=value flags onto target.
func ApplySetFlags(target map[string]any, setVals []string) error {
	for _, setVal := range setVals {
		parts := strings.SplitN(setVal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set flag %q: expected key=value format", setVal)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("invalid --set flag %q: key cannot be empty", setVal)
		}
		val := strings.TrimSpace(parts[1])
		parsedVal := parseSetVal(val)
		setPath(target, strings.Split(key, "."), parsedVal)
	}
	return nil
}

// ApplySetJSONFlags parses and applies --set-json key=json_or_yaml_value flags onto target.
func ApplySetJSONFlags(target map[string]any, setJSONVals []string) error {
	for _, setJSONVal := range setJSONVals {
		parts := strings.SplitN(setJSONVal, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid --set-json flag %q: expected key=value format", setJSONVal)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return fmt.Errorf("invalid --set-json flag %q: key cannot be empty", setJSONVal)
		}
		rawVal := strings.TrimSpace(parts[1])
		var parsedVal any
		if err := yaml.Unmarshal([]byte(rawVal), &parsedVal); err != nil {
			return fmt.Errorf("invalid json/yaml in --set-json %q: %w", setJSONVal, err)
		}
		setPath(target, strings.Split(key, "."), cleanYAMLValue(parsedVal))
	}
	return nil
}

func setPath(target map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		if existingMap, ok := target[path[0]].(map[string]any); ok {
			if newMap, ok := value.(map[string]any); ok {
				for k, v := range newMap {
					existingMap[k] = v
				}
				return
			}
		}
		target[path[0]] = value
		return
	}

	currKey := path[0]
	subMap, ok := target[currKey].(map[string]any)
	if !ok {
		subMap = make(map[string]any)
		target[currKey] = subMap
	}
	setPath(subMap, path[1:], value)
}

func parseSetVal(val string) any {
	trimmed := strings.TrimSpace(val)
	// Explicit map syntax: {k1=v1,k2=v2} or {k1: v1, k2: v2}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner == "" {
			return make(map[string]any)
		}
		// Check comma-separated k=v pairs: {k1=v1,k2=v2}
		if strings.Contains(inner, "=") {
			pairs := strings.Split(inner, ",")
			m := make(map[string]any)
			validMap := true
			for _, pair := range pairs {
				k, v, ok := strings.Cut(pair, "=")
				if !ok || strings.TrimSpace(k) == "" {
					validMap = false
					break
				}
				m[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
			if validMap && len(m) > 0 {
				return m
			}
		}
		// Otherwise try JSON/YAML map: {k1: v1, k2: v2}
		var parsed any
		if err := yaml.Unmarshal([]byte(trimmed), &parsed); err == nil {
			if m, ok := parsed.(map[string]any); ok {
				return cleanYAMLValue(m)
			}
		}
	}
	return val
}

func cleanYAMLValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[k] = cleanYAMLValue(item)
		}
		return res
	case map[any]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[fmt.Sprintf("%v", k)] = cleanYAMLValue(item)
		}
		return res
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			res[i] = cleanYAMLValue(item)
		}
		return res
	default:
		return val
	}
}

