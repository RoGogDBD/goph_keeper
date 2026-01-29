package store

import "strings"

func ParseMeta(input string) map[string]string {
	result := map[string]string{}
	if strings.TrimSpace(input) == "" {
		return result
	}

	pairs := strings.Split(input, ",")
	for _, p := range pairs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, "=", 2)
		key := strings.TrimSpace(kv[0])
		if key == "" {
			continue
		}
		val := ""
		if len(kv) > 1 {
			val = strings.TrimSpace(kv[1])
		}
		result[key] = val
	}

	return result
}
