package util

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
)

// CompareConfigs compares two config maps and returns an error if they differ.
func CompareConfigs(config1, config2 map[string]string, exclude []string) error {
	if exclude == nil {
		exclude = []string{}
	}

	delta := []string{}
	for key, value := range config1 {
		if slices.Contains(exclude, key) {
			continue
		}

		if config2[key] != value {
			delta = append(delta, key)
		}
	}
	for key, value := range config2 {
		if slices.Contains(exclude, key) {
			continue
		}

		if config1[key] != value {
			present := slices.Contains(delta, key)
			if !present {
				delta = append(delta, key)
			}
		}
	}

	sort.Strings(delta)
	if len(delta) > 0 {
		return fmt.Errorf("different values for keys: %s", strings.Join(delta, ", "))
	}

	return nil
}

// CopyConfig creates a new map with a copy of the given config.
func CopyConfig(config map[string]string) map[string]string {
	newConfig := make(map[string]string, len(config))
	maps.Copy(newConfig, config)

	return newConfig
}
