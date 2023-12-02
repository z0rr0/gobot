package db

import (
	"encoding/json"
	"fmt"
	"sort"
)

func setToString(set map[string]struct{}) (string, error) {
	if len(set) == 0 {
		return "", nil
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	b, err := json.Marshal(&keys)
	if err != nil {
		return "", fmt.Errorf("failed to marshal set: %w", err)
	}

	return string(b), nil
}

func stringToSet(s string) (map[string]struct{}, error) {
	if s == "" {
		return nil, nil
	}

	var keys []string
	err := json.Unmarshal([]byte(s), &keys)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal set: %w", err)
	}

	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}

	return set, nil
}
