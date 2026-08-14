package main

import (
	"os"
	"path/filepath"
	"time"
)

const (
	HarnessPi     = "pi"
	HarnessClaude = "claude"
)

const ActiveWindow = 2 * time.Hour

// Sessions run from scratch dirs are prompt-eval fixtures, not work. Hidden unless asked for.
var TempCwdPrefixes = []string{"/private/var/folders", "/var/folders", "/private/tmp", "/tmp"}

// Slash-command output, hook injections, and system reminders are not prompts.
var NoisePrefixes = []string{"<", "#", "Caveat:", "[Request interrupted"}

func piSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pi", "agent", "sessions")
}

func claudeSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// Usage counts tokens by category.
type Usage struct {
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheRead  int64 `json:"cache_read"`
	CacheWrite int64 `json:"cache_write"`
}

// Total sums all four categories.
func (u Usage) Total() int64 {
	return u.Input + u.Output + u.CacheRead + u.CacheWrite
}

// Any reports whether any category is nonzero.
func (u Usage) Any() bool {
	return u.Input != 0 || u.Output != 0 || u.CacheRead != 0 || u.CacheWrite != 0
}

// Add accumulates other into u.
func (u *Usage) Add(other Usage) {
	u.Input += other.Input
	u.Output += other.Output
	u.CacheRead += other.CacheRead
	u.CacheWrite += other.CacheWrite
}

// Block is one normalized content block of a message. Type is either "text"
// or "tool_use"; the other fields are set per type. Tool identity is
// normalized to the Claude spelling across harnesses (see normalize.go), so a
// consumer never has to know which harness produced it. thinking and
// tool_result blocks are dropped during normalization, matching archive-session.
type Block struct {
	Type  string         `json:"type"`            // "text" | "tool_use"
	Text  string         `json:"text,omitempty"`  // for "text"
	Name  string         `json:"name,omitempty"`  // for "tool_use" (canonical)
	Input map[string]any `json:"input,omitempty"` // for "tool_use"
}

// Event is one message-level entry from a transcript.
type Event struct {
	TS     time.Time // zero value means "no timestamp"
	TSRaw  string    // the timestamp as recorded (ISO, usually UTC); "" if none
	UUID   string    // per-message id (claude "uuid", pi entry "id"); note delta key
	Meta   bool      // claude isMeta: a system-injected turn, not conversation
	Model  string
	Usage  Usage
	Cost   *float64 // recorded cost (pi only); nil if absent
	Role   string   // "user" or "assistant"
	Text   string   // first text block of the message content
	Blocks []Block  // full normalized content (text + tool_use)
}

// Session is one parsed transcript file.
type Session struct {
	Harness     string
	ID          string
	Path        string
	CWD         string
	Name        string // resolved display title: CustomTitle || AITitle || ""
	CustomTitle string // raw user/session title; "" if absent
	AITitle     string // raw generated title (claude only); "" if absent
	Slug        string // claude only; "" if absent
	Version     string // harness version string; "" if absent
	Provider    string // last provider seen; "" if absent
	GitBranch   string // "" if the transcript carries no branch
	Events      []Event
}

// Row is one session-day: a session's activity within a single calendar day.
type Row struct {
	Date     time.Time // local midnight, from dayOf
	Harness  string
	ID       string
	Name     string
	Project  string
	CWD      string
	Start    time.Time
	End      time.Time
	Models   []string // short model names, order of first use
	Tokens   int64
	Usage    Usage
	Cost     float64
	Priced   bool     // false if any event used an unpriced model
	Unpriced []string // raw model names (pre-shortModel) that had no pricing entry
	Messages int
	Active   bool
	Path     string
}

// dayOf truncates t to local midnight. All "date" values in this program are
// produced by this function so they compare with ==.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}
