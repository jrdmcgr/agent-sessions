package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"
)

// usageText is printed for -h/--help. Content mirrors the Python argparse
// help closely enough to be useful; exact wording is ours to choose.
const usageText = `usage: sessions [-h] [--week [OFFSET] | --yesterday | --all] [--since YYYY-MM-DD]
                 [--until YYYY-MM-DD] [--project SUBSTR] [--harness {pi,claude}]
                 [--active] [--temp] [--json] [date]

List pi and Claude Code sessions worked on in a date range.

positional arguments:
  date                  a single day (YYYY-MM-DD)

options:
  -h, --help            show this help message and exit
  --week [OFFSET]       calendar week, Monday-based; OFFSET -1 is last week
  --yesterday           yesterday only
  --all                 every session on disk
  --since YYYY-MM-DD
  --until YYYY-MM-DD
  --project SUBSTR      filter by cwd/repo substring
  --harness {pi,claude}  filter by harness
  --active              only sessions touched in the last 2h
  --temp                include sessions run from temp dirs (prompt-eval fixtures)
  --json                emit JSON instead of a table

Defaults to today. Rows are session-days; totals sum the range.
`

// jsonRow is the on-the-wire shape for --json output. Field order here is
// the emitted key order.
type jsonRow struct {
	Date            string   `json:"date"`
	Harness         string   `json:"harness"`
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Project         string   `json:"project"`
	CWD             string   `json:"cwd"`
	Start           string   `json:"start"`
	End             string   `json:"end"`
	Models          []string `json:"models"`
	Tokens          int64    `json:"tokens"`
	Usage           Usage    `json:"usage"`
	Cost            float64  `json:"cost"`
	Priced          bool     `json:"priced"`
	Messages        int      `json:"messages"`
	Active          bool     `json:"active"`
	Path            string   `json:"path"`
	DurationMinutes int64    `json:"duration_minutes"`
}

// run executes the program against explicit roots and writers so tests can
// inject fixtures. Returns an exit code.
func run(argv []string, piRoot, claudeRoot string, stdout, stderr io.Writer, now time.Time) int {
	opts, err := parseArgs(argv)
	if err != nil {
		if err == errHelp {
			fmt.Fprint(stdout, usageText)
			return 0
		}
		fmt.Fprintln(stderr, err)
		return 2
	}

	start, end := resolveRange(opts, now)

	var days map[time.Time]bool
	if !start.IsZero() {
		days = map[time.Time]bool{}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			days[d] = true
		}
	}

	files := discoverIn(piRoot, claudeRoot, readPiSession, readClaudeSession, start)

	var rows []Row
	for _, f := range files {
		s := f.reader(f.path)
		if s == nil {
			continue
		}
		if opts.harness != "" && s.Harness != opts.harness {
			continue
		}
		if opts.project != "" && !strings.Contains(strings.ToLower(s.CWD), strings.ToLower(opts.project)) {
			continue
		}
		if !opts.temp && hasTempPrefix(s.CWD) {
			continue
		}
		for _, row := range sessionDays(s, days, now) {
			if row.Start.IsZero() {
				continue
			}
			if opts.active && !row.Active {
				continue
			}
			rows = append(rows, row)
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].Date.Equal(rows[j].Date) {
			return rows[i].Date.Before(rows[j].Date)
		}
		return rows[i].Start.Before(rows[j].Start)
	})

	if opts.jsonOut {
		writeJSON(stdout, rows)
		return 0
	}

	var header string
	switch {
	case start.IsZero():
		header = "all sessions"
	case start.Equal(end):
		header = start.Format("Monday 2006-01-02")
	default:
		header = start.Format("2006-01-02") + " .. " + end.Format("2006-01-02")
	}
	fmt.Fprintln(stdout, header)
	fmt.Fprintln(stdout)

	showDate := distinctDates(rows) > 1 || (days != nil && len(days) > 1)
	renderTable(stdout, stderr, rows, showDate)

	return 0
}

// hasTempPrefix reports whether cwd starts with any TempCwdPrefixes entry.
func hasTempPrefix(cwd string) bool {
	for _, prefix := range TempCwdPrefixes {
		if strings.HasPrefix(cwd, prefix) {
			return true
		}
	}
	return false
}

// distinctDates counts the number of distinct Row.Date values.
func distinctDates(rows []Row) int {
	seen := map[time.Time]bool{}
	for _, r := range rows {
		seen[r.Date] = true
	}
	return len(seen)
}

// writeJSON writes rows as a JSON array (2-space indent) followed by a
// trailing newline. Empty rows -> "[]".
func writeJSON(w io.Writer, rows []Row) {
	out := make([]jsonRow, len(rows))
	for i, r := range rows {
		models := r.Models
		if models == nil {
			models = []string{}
		}
		out[i] = jsonRow{
			Date:            r.Date.Format("2006-01-02"),
			Harness:         r.Harness,
			ID:              r.ID,
			Name:            r.Name,
			Project:         r.Project,
			CWD:             r.CWD,
			Start:           r.Start.Format("2006-01-02T15:04:05"),
			End:             r.End.Format("2006-01-02T15:04:05"),
			Models:          models,
			Tokens:          r.Tokens,
			Usage:           r.Usage,
			Cost:            r.Cost,
			Priced:          r.Priced,
			Messages:        r.Messages,
			Active:          r.Active,
			Path:            r.Path,
			DurationMinutes: int64(r.End.Sub(r.Start) / time.Minute),
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return
	}
}

// epipeWriter swallows broken-pipe write errors so `sessions | head` doesn't
// print a panic/error. Go's os.Stdout.Write returns a plain error (not a
// signal) on EPIPE, so unlike Python we don't need a signal handler — just
// ignore the write error here. Divergence from Python: we do not exit 130 on
// SIGINT; Go's default ^C behavior (process death, no special exit code
// handling) is accepted as-is per the task spec.
type epipeWriter struct {
	w io.Writer
}

func (e epipeWriter) Write(p []byte) (int, error) {
	n, err := e.w.Write(p)
	if err != nil && errors.Is(err, syscall.EPIPE) {
		return len(p), nil
	}
	return n, err
}

func main() {
	os.Exit(run(os.Args[1:], piSessionsDir(), claudeSessionsDir(), epipeWriter{os.Stdout}, os.Stderr, time.Now()))
}
