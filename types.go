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
	// Subagents holds every Task/Agent subagent Claude Code spawned during
	// this session, read from the sibling "<session-id>/subagents/"
	// directory. nil for pi sessions and Claude sessions that spawned none.
	// Their cost/usage is not part of Events; consumers that want an
	// accurate total must fold Subagents in themselves (sessionDays and
	// buildRecord both do).
	Subagents []SubagentTranscript
}

// SubagentTranscript is one Task/Agent subagent spawned by a Claude Code
// session (Claude Code writes each one to its own transcript file rather
// than inlining it, unlike the older isSidechain-in-main-file shape). Cost
// here is real spend the parent session caused but that its own transcript
// never records.
type SubagentTranscript struct {
	ID          string // agent id, from the "agent-<id>.jsonl" filename
	Name        string // meta.json "name" (the spawning tool call's label); "" if absent
	Description string // meta.json "description"; "" if absent
	AgentType   string // meta.json "agentType" (e.g. "fork", "general-purpose", "Explore"); "" if absent
	Path        string
	Models      []string // raw model ids, order of first use
	Usage       Usage
	Cost        float64
	Priced      bool     // false if any turn used an unpriced model
	Unpriced    []string // raw model names with no pricing entry
	Messages    int
	Start, End  time.Time
}

// Row is one session-day: a session's activity within a single calendar day.
// Usage/Cost/Priced/Unpriced are totals including any subagent activity that
// fell on this day (see Subagent* below for the isolated portion) — the
// number someone scanning the table for "what did this cost me" wants by
// default.
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
	Priced   bool     // false if any event or subagent used an unpriced model
	Unpriced []string // raw model names (pre-shortModel) that had no pricing entry
	Messages int
	Active   bool
	Path     string

	// SubagentUsage/SubagentCost/SubagentPriced/SubagentUnpriced isolate the
	// portion of the totals above that came from spawned subagents (Task
	// tool calls), so a consumer can report "$X, $Y of it subagents" instead
	// of a single merged number that hides where the spend went.
	SubagentUsage    Usage
	SubagentCost     float64
	SubagentPriced   bool
	SubagentUnpriced []string
}

// dayOf truncates t to local midnight. All "date" values in this program are
// produced by this function so they compare with ==.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}
