package util

import "fmt"

func GetString(m map[string]interface{}, key string) (string, error) {
	v := m[key]
	if v == nil {
		return "", fmt.Errorf("%s is required", key)
	}

	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s should be string", key)
	}

	return s, nil
}

func GetBool(m map[string]interface{}, key string) (bool, error) {
	v := m[key]
	if v == nil {
		return false, fmt.Errorf("%s is required", key)
	}

	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("%s should be bool", key)
	}

	return b, nil
}
