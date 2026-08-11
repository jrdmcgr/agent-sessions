package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildMainFixtures creates a temp pi/claude root pair with:
//   - a pi transcript in /Users/jared/proj-pi with events on 2026-08-04 and
//     2026-08-05 (two session-days).
//   - a claude transcript in /Users/jared/proj-claude with an event on
//     2026-08-05 only.
//   - a pi transcript whose cwd is under /tmp (temp-dir filtering target).
//   - a claude transcript in a different project dir, /Users/jared/other-proj
//     (project-filter target), with an event on 2026-08-05.
//
// All files get their mtimes set to 2026-08-05 so discovery's cutoff never
// excludes them regardless of when the test runs.
func buildMainFixtures(t *testing.T) (piRoot, claudeRoot string) {
	t.Helper()
	tmp := t.TempDir()
	piRoot = filepath.Join(tmp, "pi")
	claudeRoot = filepath.Join(tmp, "claude")

	piContent := `{"type":"session","cwd":"/Users/jared/proj-pi","id":"pi-sess-1","timestamp":"2026-08-04T10:00:00Z"}
{"type":"model_change","modelId":"claude-opus-4","timestamp":"2026-08-04T10:00:00Z"}
{"type":"message","timestamp":"2026-08-04T10:00:00Z","message":{"role":"user","content":"day one work"}}
{"type":"message","timestamp":"2026-08-04T10:05:00Z","message":{"role":"assistant","model":"claude-opus-4","content":"reply one","usage":{"input":100,"output":50,"cacheRead":0,"cacheWrite":0,"cost":{"total":0.01}}}}
{"type":"message","timestamp":"2026-08-05T09:00:00Z","message":{"role":"user","content":"day two work"}}
{"type":"message","timestamp":"2026-08-05T09:10:00Z","message":{"role":"assistant","model":"claude-opus-4","content":"reply two","usage":{"input":200,"output":75,"cacheRead":0,"cacheWrite":0,"cost":{"total":0.02}}}}
`
	piPath := writePiFixture(t, piRoot, "-Users-jared-proj-pi", "2026-08-04_pi-sess-1.jsonl", piContent)

	claudeContent := `{"type":"user","cwd":"/Users/jared/proj-claude","sessionId":"claude-sess-1","timestamp":"2026-08-05T11:00:00Z","message":{"content":"claude day two"}}
{"type":"assistant","cwd":"/Users/jared/proj-claude","sessionId":"claude-sess-1","timestamp":"2026-08-05T11:05:00Z","message":{"model":"claude-sonnet-4-5","content":[{"type":"text","text":"claude reply"}],"usage":{"input_tokens":300,"output_tokens":100,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}
`
	claudePath := writeClaudeFixture(t, claudeRoot, "-Users-jared-proj-claude", "claude-sess-1.jsonl", claudeContent)

	tempContent := `{"type":"session","cwd":"/tmp/scratch-eval","id":"tmp-sess-1","timestamp":"2026-08-05T12:00:00Z"}
{"type":"message","timestamp":"2026-08-05T12:00:00Z","message":{"role":"user","content":"temp fixture work"}}
`
	tempPath := writePiFixture(t, piRoot, "-tmp-scratch-eval", "2026-08-05_tmp-sess-1.jsonl", tempContent)

	otherContent := `{"type":"user","cwd":"/Users/jared/other-proj","sessionId":"claude-sess-2","timestamp":"2026-08-05T13:00:00Z","message":{"content":"other project work"}}
{"type":"assistant","cwd":"/Users/jared/other-proj","sessionId":"claude-sess-2","timestamp":"2026-08-05T13:05:00Z","message":{"model":"claude-sonnet-4-5","content":"other reply","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}
`
	otherPath := writeClaudeFixture(t, claudeRoot, "-Users-jared-other-proj", "claude-sess-2.jsonl", otherContent)

	mtime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.Local)
	for _, p := range []string{piPath, claudePath, tempPath, otherPath} {
		if err := os.Chtimes(p, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	return piRoot, claudeRoot
}

// fixedNow is used as `now` across the main_test.go tests; it is well after
// all fixture timestamps so nothing is spuriously "active" unless a test
// wants it to be.
var fixedNow = time.Date(2026, 8, 6, 0, 0, 0, 0, time.Local)

func TestRunDefaultSingleDay(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"2026-08-05"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	out := stdout.String()
	if !strings.HasPrefix(out, "Wednesday 2026-08-05\n\n") {
		t.Fatalf("header wrong, got:\n%s", out)
	}
	if strings.Contains(out, "DATE") {
		t.Errorf("single-day, single-date result set should not show a DATE column:\n%s", out)
	}
	// The 08-04 day of the pi session should not appear (it's a different
	// day), and neither should the /tmp or other-proj sessions (filtered
	// separately, but also not part of 08-05... wait, other-proj IS on
	// 08-05, so it should appear; /tmp should not, by default).
	if strings.Contains(out, "scratch-eval") || strings.Contains(out, "tmp-sess-1") {
		t.Errorf("temp session should be excluded by default:\n%s", out)
	}
	if !strings.Contains(out, "proj-pi") {
		t.Errorf("expected proj-pi row for 08-05:\n%s", out)
	}
	if !strings.Contains(out, "proj-claude") {
		t.Errorf("expected proj-claude row for 08-05:\n%s", out)
	}
}

func TestRunAllJSON(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--all", "--json"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\noutput:\n%s", err, stdout.String())
	}

	// pi session contributes 2 session-days, claude proj-claude 1, claude
	// other-proj 1. /tmp session excluded. Total: 4.
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4: %+v", len(rows), rows)
	}

	for _, r := range rows {
		for _, key := range []string{
			"date", "harness", "id", "name", "project", "cwd", "start", "end",
			"models", "tokens", "usage", "cost", "priced", "messages", "active",
			"path", "duration_minutes",
		} {
			if _, ok := r[key]; !ok {
				t.Errorf("row missing key %q: %+v", key, r)
			}
		}
		date, _ := r["date"].(string)
		if _, err := time.Parse("2006-01-02", date); err != nil {
			t.Errorf("date %q not in 2006-01-02 form: %v", date, err)
		}
		start, _ := r["start"].(string)
		if _, err := time.Parse("2006-01-02T15:04:05", start); err != nil {
			t.Errorf("start %q not in expected form: %v", start, err)
		}
	}

	found := false
	for _, r := range rows {
		if strings.Contains(r["cwd"].(string), "scratch-eval") {
			found = true
		}
	}
	if found {
		t.Error("/tmp session should be absent from --all --json without --temp")
	}

	// duration_minutes sanity: the pi 08-04 row spans 10:00:00 -> 10:05:00.
	for _, r := range rows {
		if r["date"] == "2026-08-04" {
			dm, ok := r["duration_minutes"].(float64)
			if !ok || dm != 5 {
				t.Errorf("08-04 row duration_minutes = %v, want 5", r["duration_minutes"])
			}
		}
	}
}

func TestRunTempAllJSON(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--temp", "--all", "--json"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	found := false
	for _, r := range rows {
		if strings.Contains(r["cwd"].(string), "scratch-eval") {
			found = true
		}
	}
	if !found {
		t.Errorf("/tmp session should be present with --temp: %+v", rows)
	}
}

// Regression (task 12 parity, 2026-08-11): a since-removed filter dropped
// any row with Name == "(unnamed)", Cost == 0, and !Active. Python has no
// such filter — a session where no work happened (started, no assistant
// reply, no model) is still a real row, just an uninformative one. --all
// must include it.
func TestRunIncludesZeroCostUnnamedInactiveSession(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)

	claudeContent := `{"type":"user","cwd":"/Users/jared/proj-claude","sessionId":"claude-empty-1","timestamp":"2026-08-05T14:00:00Z","message":{"content":"<system-reminder>noise</system-reminder>"}}
{"type":"user","cwd":"/Users/jared/proj-claude","sessionId":"claude-empty-1","timestamp":"2026-08-05T14:00:01Z","message":{"content":"<system-reminder>still noise</system-reminder>"}}
`
	emptyPath := writeClaudeFixture(t, claudeRoot, "-Users-jared-proj-claude", "claude-empty-1.jsonl", claudeContent)
	mtime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.Local)
	if err := os.Chtimes(emptyPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--all", "--json"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	found := false
	for _, r := range rows {
		if r["id"] == "claude-empty-1" {
			found = true
			if r["name"] != "(unnamed)" {
				t.Errorf("name = %v, want (unnamed)", r["name"])
			}
			if r["cost"] != 0.0 {
				t.Errorf("cost = %v, want 0", r["cost"])
			}
			if r["active"] != false {
				t.Errorf("active = %v, want false", r["active"])
			}
		}
	}
	if !found {
		t.Error("zero-cost unnamed inactive session was dropped; python has no such filter")
	}
}

func TestRunProjectFilter(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--all", "--json", "--project", "other-proj"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	if !strings.Contains(rows[0]["cwd"].(string), "other-proj") {
		t.Errorf("unexpected row: %+v", rows[0])
	}
}

func TestRunHarnessFilter(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--all", "--json", "--harness", "pi"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (both pi session-days): %+v", len(rows), rows)
	}
	for _, r := range rows {
		if r["harness"] != "pi" {
			t.Errorf("unexpected harness in filtered output: %+v", r)
		}
	}
}

func TestRunUnknownFlag(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--bogus"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.String() == "" {
		t.Error("expected an error message on stderr")
	}
}

func TestRunHelp(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--help"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout.Len() == 0 {
		t.Error("expected usage text on stdout")
	}
}

// TestRunJSONKeyOrder guards the emitted key order in raw JSON text: "date"
// must precede "harness" must precede "id" in the first row.
func TestRunJSONKeyOrder(t *testing.T) {
	piRoot, claudeRoot := buildMainFixtures(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{"--all", "--json"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}

	out := stdout.String()
	dateIdx := strings.Index(out, `"date"`)
	harnessIdx := strings.Index(out, `"harness"`)
	idIdx := strings.Index(out, `"id"`)
	if dateIdx < 0 || harnessIdx < 0 || idIdx < 0 {
		t.Fatalf("missing expected keys in output:\n%s", out)
	}
	if !(dateIdx < harnessIdx && harnessIdx < idIdx) {
		t.Errorf("key order wrong: date=%d harness=%d id=%d\n%s", dateIdx, harnessIdx, idIdx, out)
	}
}

func TestRunEmptyRowsJSON(t *testing.T) {
	tmp := t.TempDir()
	piRoot := filepath.Join(tmp, "pi")
	claudeRoot := filepath.Join(tmp, "claude")
	var stdout, stderr bytes.Buffer

	code := run([]string{"--all", "--json"}, piRoot, claudeRoot, &stdout, &stderr, fixedNow)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "[]" {
		t.Errorf("empty result set should print []: %q", stdout.String())
	}
}
