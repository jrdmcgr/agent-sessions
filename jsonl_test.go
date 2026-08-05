package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIterJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	content := "{\"a\":1}\n\n{not json\n\"hi\"\n{\"b\":2}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	got := iterJSONL(path)
	if len(got) != 2 {
		t.Fatalf("expected 2 objects, got %d: %v", len(got), got)
	}
	if v, ok := got[0]["a"].(float64); !ok || v != 1 {
		t.Errorf("first object = %v, want a=1", got[0])
	}
	if v, ok := got[1]["b"].(float64); !ok || v != 2 {
		t.Errorf("second object = %v, want b=2", got[1])
	}
}

func TestIterJSONLNonexistent(t *testing.T) {
	got := iterJSONL("/nonexistent/path/that/should/not/exist.jsonl")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestParseTS(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name string
		in   any
		want time.Time
	}{
		{
			name: "z-suffixed with millis",
			in:   "2026-08-05T14:03:22.123Z",
			want: time.Date(2026, 8, 5, 14, 3, 22, 123000000, time.UTC).In(loc),
		},
		{
			name: "offset form",
			in:   "2026-08-05T14:03:22+02:00",
			want: time.Date(2026, 8, 5, 14, 3, 22, 0, time.FixedZone("+02:00", 2*3600)).In(loc),
		},
		{
			name: "naive form",
			in:   "2026-08-05T14:03:22",
			want: time.Date(2026, 8, 5, 14, 3, 22, 0, loc),
		},
		{name: "nil", in: nil, want: time.Time{}},
		{name: "empty string", in: "", want: time.Time{}},
		{name: "not a date", in: "not-a-date", want: time.Time{}},
		{name: "integer", in: 5, want: time.Time{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTS(tc.in)
			if !got.Equal(tc.want) {
				t.Errorf("parseTS(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFirstText(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{
			name: "plain string",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "list with tool_use before text",
			in: []any{
				map[string]any{"type": "tool_use", "text": "should not use this"},
				map[string]any{"type": "text", "text": "the actual text"},
			},
			want: "the actual text",
		},
		{
			name: "list with no text block",
			in: []any{
				map[string]any{"type": "tool_use"},
				map[string]any{"type": "image"},
			},
			want: "",
		},
		{
			name: "nil",
			in:   nil,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := firstText(tc.in)
			if got != tc.want {
				t.Errorf("firstText(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"present":   "value",
		"empty":     "",
		"wrongtype": 42,
	}
	if got := getString(m, "present"); got != "value" {
		t.Errorf("present: got %q, want %q", got, "value")
	}
	if got := getString(m, "absent"); got != "" {
		t.Errorf("absent: got %q, want empty", got)
	}
	if got := getString(m, "wrongtype"); got != "" {
		t.Errorf("wrongtype: got %q, want empty", got)
	}
}

func TestGetMap(t *testing.T) {
	inner := map[string]any{"x": 1.0}
	m := map[string]any{
		"present":   inner,
		"wrongtype": "not a map",
	}
	got := getMap(m, "present")
	if got == nil || got["x"] != 1.0 {
		t.Errorf("present: got %v, want %v", got, inner)
	}
	if got := getMap(m, "absent"); got != nil {
		t.Errorf("absent: got %v, want nil", got)
	}
	if got := getMap(m, "wrongtype"); got != nil {
		t.Errorf("wrongtype: got %v, want nil", got)
	}
}

func TestGetInt64(t *testing.T) {
	m := map[string]any{
		"present":   float64(42),
		"wrongtype": "not a number",
	}
	if got := getInt64(m, "present"); got != 42 {
		t.Errorf("present: got %d, want 42", got)
	}
	if got := getInt64(m, "absent"); got != 0 {
		t.Errorf("absent: got %d, want 0", got)
	}
	if got := getInt64(m, "wrongtype"); got != 0 {
		t.Errorf("wrongtype: got %d, want 0", got)
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{
		"present":   true,
		"wrongtype": "not a bool",
	}
	if got := getBool(m, "present"); got != true {
		t.Errorf("present: got %v, want true", got)
	}
	if got := getBool(m, "absent"); got != false {
		t.Errorf("absent: got %v, want false", got)
	}
	if got := getBool(m, "wrongtype"); got != false {
		t.Errorf("wrongtype: got %v, want false", got)
	}
}
