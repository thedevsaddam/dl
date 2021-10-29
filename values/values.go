package values

import "strings"

// MapStrSliceString maps a string key to a list of MapStrSliceString.
type MapStrSliceString map[string][]string

// Get return the keys name based on extension.
func (v MapStrSliceString) Get(ext string) string {
	if v == nil {
		return ""
	}
	for k, extensions := range v {
		for _, e := range extensions {
			if strings.ToLower(strings.TrimSpace(e)) == strings.ToLower(strings.TrimSpace(ext)) {
				return k
			}
		}
	}
	return ""
}

// Add adds the value to key. It appends to any existing
// MapStrSliceString associated with key.
func (v MapStrSliceString) Add(key, value string) {
	for _, e := range v[key] {
		if strings.ToLower(strings.TrimSpace(e)) == strings.ToLower(strings.TrimSpace(value)) {
			return
		}
	}
	v[key] = append(v[key], value)
}
