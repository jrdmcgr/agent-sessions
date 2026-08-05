package main

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func TestParseDay(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := parseDay("2026-08-05")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("invalid calendar date", func(t *testing.T) {
		_, err := parseDay("2026-02-30")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("garbage", func(t *testing.T) {
		_, err := parseDay("not-a-date")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		want := `expected YYYY-MM-DD, got "not-a-date"`
		if err.Error() != want {
			t.Errorf("got %q, want %q", err.Error(), want)
		}
	})
}

func TestWeekBounds(t *testing.T) {
	wed := day(2026, 8, 5) // Wednesday
	monday := day(2026, 8, 3)
	sunday := day(2026, 8, 9)

	tests := []struct {
		name      string
		offset    int
		today     time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{"wed offset 0", 0, wed, monday, sunday},
		{"monday offset -1", -1, monday, day(2026, 7, 27), day(2026, 8, 2)},
		{
			// A Sunday input stays in ITS week: the Monday 6 days earlier.
			"sunday stays in its own week", 0, day(2026, 8, 9),
			day(2026, 8, 3), day(2026, 8, 9),
		},
		{"offset +1 from wednesday", 1, wed, day(2026, 8, 10), day(2026, 8, 16)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := weekBounds(tc.offset, tc.today)
			if !start.Equal(tc.wantStart) || !end.Equal(tc.wantEnd) {
				t.Errorf("weekBounds(%d, %v) = (%v, %v), want (%v, %v)",
					tc.offset, tc.today, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestResolveRange(t *testing.T) {
	today := day(2026, 8, 5) // Wednesday

	weekZero := 0
	weekMinus1 := -1

	tests := []struct {
		name      string
		opts      options
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "all wins over everything",
			opts:      options{all: true, date: day(2026, 1, 1)},
			wantStart: time.Time{},
			wantEnd:   time.Time{},
		},
		{
			name:      "positional date",
			opts:      options{date: day(2026, 7, 24)},
			wantStart: day(2026, 7, 24),
			wantEnd:   day(2026, 7, 24),
		},
		{
			name:      "since and until both set",
			opts:      options{since: day(2026, 7, 1), until: day(2026, 7, 15)},
			wantStart: day(2026, 7, 1),
			wantEnd:   day(2026, 7, 15),
		},
		{
			name:      "since alone gets until=today",
			opts:      options{since: day(2026, 7, 1)},
			wantStart: day(2026, 7, 1),
			wantEnd:   today,
		},
		{
			name:      "until alone gets since=1970-01-01",
			opts:      options{until: day(2026, 7, 15)},
			wantStart: day(1970, 1, 1),
			wantEnd:   day(2026, 7, 15),
		},
		{
			name:      "week non-nil, offset 0",
			opts:      options{week: &weekZero},
			wantStart: day(2026, 8, 3),
			wantEnd:   day(2026, 8, 9),
		},
		{
			name:      "week non-nil, offset -1",
			opts:      options{week: &weekMinus1},
			wantStart: day(2026, 7, 27),
			wantEnd:   day(2026, 8, 2),
		},
		{
			name:      "yesterday",
			opts:      options{yesterday: true},
			wantStart: day(2026, 8, 4),
			wantEnd:   day(2026, 8, 4),
		},
		{
			name:      "default is today",
			opts:      options{},
			wantStart: today,
			wantEnd:   today,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := resolveRange(&tc.opts, today)
			if !start.Equal(tc.wantStart) || !end.Equal(tc.wantEnd) {
				t.Errorf("resolveRange(%+v, %v) = (%v, %v), want (%v, %v)",
					tc.opts, today, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	t.Run("flag value forms", func(t *testing.T) {
		o, err := parseArgs([]string{"--project", "foo"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.project != "foo" {
			t.Errorf("got project %q, want %q", o.project, "foo")
		}

		o, err = parseArgs([]string{"--project=bar"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.project != "bar" {
			t.Errorf("got project %q, want %q", o.project, "bar")
		}
	})

	t.Run("bare --week defaults to 0", func(t *testing.T) {
		o, err := parseArgs([]string{"--week"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.week == nil || *o.week != 0 {
			t.Errorf("got week %v, want 0", o.week)
		}
	})

	t.Run("--week -1 consumes the integer", func(t *testing.T) {
		o, err := parseArgs([]string{"--week", "-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.week == nil || *o.week != -1 {
			t.Errorf("got week %v, want -1", o.week)
		}
	})

	t.Run("--week=-1", func(t *testing.T) {
		o, err := parseArgs([]string{"--week=-1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.week == nil || *o.week != -1 {
			t.Errorf("got week %v, want -1", o.week)
		}
	})

	t.Run("--week followed by non-integer flag is not consumed", func(t *testing.T) {
		o, err := parseArgs([]string{"--week", "--json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.week == nil || *o.week != 0 {
			t.Errorf("got week %v, want 0", o.week)
		}
		if !o.jsonOut {
			t.Error("expected --json to still be parsed")
		}
	})

	t.Run("--week followed by a date is not consumed", func(t *testing.T) {
		o, err := parseArgs([]string{"--week", "2026-08-05"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.week == nil || *o.week != 0 {
			t.Errorf("got week %v, want 0", o.week)
		}
		if o.date.IsZero() {
			t.Error("expected positional date to still be parsed")
		}
	})

	t.Run("single positional date", func(t *testing.T) {
		o, err := parseArgs([]string{"2026-07-24"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !o.date.Equal(day(2026, 7, 24)) {
			t.Errorf("got date %v, want %v", o.date, day(2026, 7, 24))
		}
	})

	t.Run("second positional is an error", func(t *testing.T) {
		_, err := parseArgs([]string{"2026-07-24", "2026-07-25"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("bad positional date is an error", func(t *testing.T) {
		_, err := parseArgs([]string{"not-a-date"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("week and yesterday are mutually exclusive", func(t *testing.T) {
		_, err := parseArgs([]string{"--week", "--yesterday"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("week and all are mutually exclusive", func(t *testing.T) {
		_, err := parseArgs([]string{"--all", "--week"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("yesterday and all are mutually exclusive", func(t *testing.T) {
		_, err := parseArgs([]string{"--yesterday", "--all"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repeating the same when-flag is fine", func(t *testing.T) {
		_, err := parseArgs([]string{"--all", "--all"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("harness pi is valid", func(t *testing.T) {
		o, err := parseArgs([]string{"--harness", "pi"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.harness != HarnessPi {
			t.Errorf("got harness %q, want %q", o.harness, HarnessPi)
		}
	})

	t.Run("harness claude is valid", func(t *testing.T) {
		o, err := parseArgs([]string{"--harness", "claude"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if o.harness != HarnessClaude {
			t.Errorf("got harness %q, want %q", o.harness, HarnessClaude)
		}
	})

	t.Run("harness invalid value is an error", func(t *testing.T) {
		_, err := parseArgs([]string{"--harness", "codex"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unknown flag is an error naming it", func(t *testing.T) {
		_, err := parseArgs([]string{"--bogus"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !containsSubstr(err.Error(), "--bogus") {
			t.Errorf("error %q does not name the offending flag", err.Error())
		}
	})

	t.Run("help flags return errHelp", func(t *testing.T) {
		_, err := parseArgs([]string{"--help"})
		if err != errHelp {
			t.Errorf("got %v, want errHelp", err)
		}
		_, err = parseArgs([]string{"-h"})
		if err != errHelp {
			t.Errorf("got %v, want errHelp", err)
		}
	})

	t.Run("boolean flags set true", func(t *testing.T) {
		o, err := parseArgs([]string{"--active", "--temp", "--json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !o.active || !o.temp || !o.jsonOut {
			t.Errorf("got %+v, want all true", o)
		}
	})

	t.Run("since and until parse as days", func(t *testing.T) {
		o, err := parseArgs([]string{"--since", "2026-07-01", "--until=2026-07-15"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !o.since.Equal(day(2026, 7, 1)) || !o.until.Equal(day(2026, 7, 15)) {
			t.Errorf("got since=%v until=%v", o.since, o.until)
		}
	})

	t.Run("bad since value is an error", func(t *testing.T) {
		_, err := parseArgs([]string{"--since", "nope"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func containsSubstr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
