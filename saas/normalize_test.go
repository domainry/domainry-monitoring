package saas_test

import "encoding/json"

func normalized(value map[string]any) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}
