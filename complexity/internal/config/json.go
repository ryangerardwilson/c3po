package config

import (
	"encoding/json"
	"os"
)

// overlayJSON merges a JSON object onto dst. Absent keys keep dst's values.
func overlayJSON(dst any, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base, err := json.Marshal(dst)
	if err != nil {
		return err
	}
	var have map[string]any
	if err := json.Unmarshal(base, &have); err != nil {
		return err
	}
	var over map[string]any
	if err := json.Unmarshal(data, &over); err != nil {
		return err
	}
	mergeMaps(have, over)
	merged, err := json.Marshal(have)
	if err != nil {
		return err
	}
	return json.Unmarshal(merged, dst)
}

func mergeMaps(dst, src map[string]any) {
	for key, val := range src {
		child, ok := val.(map[string]any)
		if !ok {
			dst[key] = val
			continue
		}
		cur, ok := dst[key].(map[string]any)
		if !ok {
			dst[key] = val
			continue
		}
		mergeMaps(cur, child)
	}
}

func writeJSON(path string, cfg Config) error {
	buf, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	buf = append(buf, '\n')
	return os.WriteFile(path, buf, 0o644)
}
