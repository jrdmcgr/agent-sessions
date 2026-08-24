package main

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// iterJSONL reads path line by line, decoding each non-blank line as a JSON
// object into map[string]any. Blank lines and lines that fail to decode are
// skipped. If the file cannot be opened, returns nil. Lines that decode to a
// non-object (e.g. a bare string) are skipped.
func iterJSONL(path string) []map[string]any {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []map[string]any
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		out = append(out, obj)
	}
	return out
}

// parseTS parses an ISO-8601 string (usually Z-suffixed, e.g.
// "2026-08-05T14:03:22.123Z", but bare "2026-08-05T14:03:22" and offset forms
// like "2026-08-05T14:03:22+02:00" must also work) and converts it to a
// time.Time in time.Local. Returns the zero time.Time for a non-string value,
// empty string, or unparseable string. A timestamp with no zone is taken as
// already local (matching Python's naive-datetime behavior).
func parseTS(value any) time.Time {
	s, ok := value.(string)
	if !ok || s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.In(time.Local)
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05.999999999", s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

// getString returns m[key] if it is a non-empty string, else "".
func getString(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

// getMap returns m[key] if it is a map[string]any, else nil.
func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

// getInt64 returns m[key] as int64 if it is a JSON number, else 0.
// (encoding/json decodes numbers into float64.)
func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

// getBool returns m[key] if it is a bool, else false.
func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// firstText extracts the user-visible text from a message content field:
//   - if content is a string, return it as-is
//   - if content is a []any, return the "text" of the first element that is a
//     map with "type" == "text" (empty string if that text is missing/nil)
//   - otherwise return ""
//
// Ports first_text from the Python.
func firstText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		for _, item := range v {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if getString(block, "type") == "text" {
				if s, ok := block["text"].(string); ok {
					return s
				}
				return ""
			}
		}
	}
	return ""
}
