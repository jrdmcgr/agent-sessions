package main

import (
	"testing"
	"time"
)

func mkTS(y int, m time.Month, d, hh, mm int) time.Time {
	return time.Date(y, m, d, hh, mm, 0, 0, time.Local)
}

func mkDay(y int, m time.Month, d int) time.Time {
	return dayOf(mkTS(y, m, d, 0, 0))
}

func ptr(f float64) *float64 { return &f }

func TestSessionDaysSplitsByDayAndFilters(t *testing.T) {
	s := &Session{
		Harness: HarnessPi,
		ID:      "abc",
		Path:    "/tmp/abc.jsonl",
		CWD:     "/Users/jared/Code/foo",
		Events: []Event{
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 10, Output: 5}, Role: "user", Text: "day1 first"},
			{TS: mkTS(2026, 1, 1, 10, 30), Model: "claude-sonnet-4-5", Usage: Usage{Input: 20, Output: 10}, Role: "assistant"},
			{TS: mkTS(2026, 1, 2, 8, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1, Output: 1}, Role: "user", Text: "day2 first"},
		},
	}
	now := mkTS(2026, 1, 2, 9, 0)

	rows := sessionDays(s, nil, now)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	day1, day2 := rows[0], rows[1]
	if !day1.Date.Equal(mkDay(2026, 1, 1)) {
		t.Errorf("row0 date = %v, want day1", day1.Date)
	}
	if !day2.Date.Equal(mkDay(2026, 1, 2)) {
		t.Errorf("row1 date = %v, want day2", day2.Date)
	}

	if !day1.Start.Equal(mkTS(2026, 1, 1, 9, 0)) {
		t.Errorf("day1 start = %v", day1.Start)
	}
	if !day1.End.Equal(mkTS(2026, 1, 1, 10, 30)) {
		t.Errorf("day1 end = %v", day1.End)
	}
	if day1.Messages != 2 {
		t.Errorf("day1 messages = %d, want 2", day1.Messages)
	}
	wantUsage := Usage{Input: 30, Output: 15}
	if day1.Usage != wantUsage {
		t.Errorf("day1 usage = %+v, want %+v", day1.Usage, wantUsage)
	}

	if day2.Messages != 1 {
		t.Errorf("day2 messages = %d, want 1", day2.Messages)
	}

	// days filter selecting only day 1
	daysFilter := map[time.Time]bool{mkDay(2026, 1, 1): true}
	filtered := sessionDays(s, daysFilter, now)
	if len(filtered) != 1 {
		t.Fatalf("filtered got %d rows, want 1", len(filtered))
	}
	if !filtered[0].Date.Equal(mkDay(2026, 1, 1)) {
		t.Errorf("filtered date = %v, want day1", filtered[0].Date)
	}
}

func TestSessionDaysSkipsZeroTS(t *testing.T) {
	s := &Session{
		CWD: "/tmp/x",
		Events: []Event{
			{Model: "claude-sonnet-4-5", Usage: Usage{Input: 1}}, // zero TS
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1}},
		},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Messages != 1 {
		t.Errorf("messages = %d, want 1 (zero-TS event should be skipped)", rows[0].Messages)
	}
}

func TestSessionDaysCostMixingPricedFalse(t *testing.T) {
	cost := 0.5
	s := &Session{
		CWD: "/tmp/x",
		Events: []Event{
			// recorded cost
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 100}, Cost: &cost},
			// priceable model, no recorded cost -> computed
			{TS: mkTS(2026, 1, 1, 9, 1), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1_000_000}},
			// unknown model, nonzero usage -> unpriced
			{TS: mkTS(2026, 1, 1, 9, 2), Model: "some-unknown-model", Usage: Usage{Input: 5}},
		},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.Priced {
		t.Errorf("priced = true, want false")
	}
	computed := price("claude-sonnet-4-5", Usage{Input: 1_000_000})
	want := cost + *computed
	if r.Cost != want {
		t.Errorf("cost = %v, want %v", r.Cost, want)
	}
}

func TestSessionDaysCostMixingPricedTrueWithoutUnknownModel(t *testing.T) {
	cost := 0.5
	s := &Session{
		CWD: "/tmp/x",
		Events: []Event{
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 100}, Cost: &cost},
			{TS: mkTS(2026, 1, 1, 9, 1), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1_000_000}},
		},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if !r.Priced {
		t.Errorf("priced = false, want true")
	}
	computed := price("claude-sonnet-4-5", Usage{Input: 1_000_000})
	want := cost + *computed
	if r.Cost != want {
		t.Errorf("cost = %v, want %v", r.Cost, want)
	}
}

func TestSessionDaysUnknownModelZeroUsageStaysPriced(t *testing.T) {
	s := &Session{
		CWD: "/tmp/x",
		Events: []Event{
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "some-unknown-model", Usage: Usage{}},
		},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if !rows[0].Priced {
		t.Errorf("priced = false, want true (all-zero usage should not affect Priced)")
	}
	if rows[0].Cost != 0 {
		t.Errorf("cost = %v, want 0", rows[0].Cost)
	}
}

func TestSessionDaysModelDedupAndOrdering(t *testing.T) {
	s := &Session{
		CWD: "/tmp/x",
		Events: []Event{
			{TS: mkTS(2026, 1, 1, 9, 0), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1}},
			{TS: mkTS(2026, 1, 1, 9, 1), Model: "", Usage: Usage{Input: 1}},
			{TS: mkTS(2026, 1, 1, 9, 2), Model: "claude-opus-4", Usage: Usage{Input: 1}},
			{TS: mkTS(2026, 1, 1, 9, 3), Model: "claude-sonnet-4-5", Usage: Usage{Input: 1}},
		},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	want := []string{"sonnet-4-5", "opus-4"}
	got := rows[0].Models
	if len(got) != len(want) {
		t.Fatalf("models = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("models[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSessionDaysNameAndProject(t *testing.T) {
	t.Run("named session uses session name on every row", func(t *testing.T) {
		s := &Session{
			Name: "my session",
			CWD:  "/Users/jared/Code/foo",
			Events: []Event{
				{TS: mkTS(2026, 1, 1, 9, 0), Role: "user", Text: "hi"},
				{TS: mkTS(2026, 1, 2, 9, 0), Role: "user", Text: "hi"},
			},
		}
		rows := sessionDays(s, nil, mkTS(2026, 1, 2, 10, 0))
		for _, r := range rows {
			if r.Name != "my session" {
				t.Errorf("name = %q, want %q", r.Name, "my session")
			}
		}
	})

	t.Run("unnamed session falls back to first user prompt", func(t *testing.T) {
		s := &Session{
			CWD: "/Users/jared/Code/foo",
			Events: []Event{
				{TS: mkTS(2026, 1, 1, 9, 0), Role: "assistant", Text: "ignored"},
				{TS: mkTS(2026, 1, 1, 9, 1), Role: "user", Text: "  fix   the\nbug  "},
			},
		}
		rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
		if rows[0].Name != "fix the bug" {
			t.Errorf("name = %q, want %q", rows[0].Name, "fix the bug")
		}
	})

	t.Run("project is base of cwd", func(t *testing.T) {
		s := &Session{
			CWD:    "/Users/jared/Code/foo",
			Events: []Event{{TS: mkTS(2026, 1, 1, 9, 0)}},
		}
		rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
		if rows[0].Project != "foo" {
			t.Errorf("project = %q, want %q", rows[0].Project, "foo")
		}
	})

	t.Run("project falls back to raw cwd when base is empty, /, or .", func(t *testing.T) {
		for _, cwd := range []string{"", "/", "."} {
			s := &Session{
				CWD:    cwd,
				Events: []Event{{TS: mkTS(2026, 1, 1, 9, 0)}},
			}
			rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
			if rows[0].Project != cwd {
				t.Errorf("cwd=%q: project = %q, want %q", cwd, rows[0].Project, cwd)
			}
		}
	})
}

func TestSessionDaysActiveFlag(t *testing.T) {
	now := mkTS(2026, 1, 1, 12, 0)

	t.Run("within active window", func(t *testing.T) {
		s := &Session{
			CWD:    "/tmp/x",
			Events: []Event{{TS: now.Add(-(1*time.Hour + 59*time.Minute))}},
		}
		rows := sessionDays(s, nil, now)
		if !rows[0].Active {
			t.Errorf("active = false, want true")
		}
	})

	t.Run("outside active window", func(t *testing.T) {
		s := &Session{
			CWD:    "/tmp/x",
			Events: []Event{{TS: now.Add(-(2*time.Hour + 1*time.Minute))}},
		}
		rows := sessionDays(s, nil, now)
		if rows[0].Active {
			t.Errorf("active = true, want false")
		}
	})
}

func TestSessionDaysCopiedFields(t *testing.T) {
	s := &Session{
		Harness: HarnessClaude,
		ID:      "sess-123",
		Path:    "/some/path.jsonl",
		CWD:     "/Users/jared/Code/bar",
		Events:  []Event{{TS: mkTS(2026, 1, 1, 9, 0)}},
	}
	rows := sessionDays(s, nil, mkTS(2026, 1, 1, 10, 0))
	r := rows[0]
	if r.Harness != HarnessClaude || r.ID != "sess-123" || r.CWD != s.CWD || r.Path != s.Path {
		t.Errorf("copied fields wrong: %+v", r)
	}
	if !r.Date.Equal(mkDay(2026, 1, 1)) {
		t.Errorf("date = %v", r.Date)
	}
}

func TestFallbackName(t *testing.T) {
	t.Run("skips assistant-first events", func(t *testing.T) {
		s := &Session{Events: []Event{
			{Role: "assistant", Text: "assistant text should be skipped"},
			{Role: "user", Text: "real prompt"},
		}}
		if got := fallbackName(s); got != "real prompt" {
			t.Errorf("got %q, want %q", got, "real prompt")
		}
	})

	t.Run("skips noise prefixes and whitespace-only", func(t *testing.T) {
		s := &Session{Events: []Event{
			{Role: "user", Text: "<system-reminder>ignore</system-reminder>"},
			{Role: "user", Text: "# header"},
			{Role: "user", Text: "Caveat: something"},
			{Role: "user", Text: "[Request interrupted by user]"},
			{Role: "user", Text: "   \n\t  "},
			{Role: "user", Text: "actual prompt here"},
		}}
		if got := fallbackName(s); got != "actual prompt here" {
			t.Errorf("got %q, want %q", got, "actual prompt here")
		}
	})

	t.Run("collapses whitespace runs", func(t *testing.T) {
		s := &Session{Events: []Event{
			{Role: "user", Text: "  fix   the\nbug  "},
		}}
		if got := fallbackName(s); got != "fix the bug" {
			t.Errorf("got %q, want %q", got, "fix the bug")
		}
	})

	t.Run("truncates to 48 bytes", func(t *testing.T) {
		prompt := "123456789012345678901234567890123456789012345678901234567890" // 60 chars
		s := &Session{Events: []Event{
			{Role: "user", Text: prompt},
		}}
		got := fallbackName(s)
		want := prompt[:48]
		if got != want {
			t.Errorf("got %q (len %d), want %q (len %d)", got, len(got), want, len(want))
		}
	})

	t.Run("all-noise session returns unnamed", func(t *testing.T) {
		s := &Session{Events: []Event{
			{Role: "assistant", Text: "ignored"},
			{Role: "user", Text: "<noise>"},
			{Role: "user", Text: "   "},
		}}
		if got := fallbackName(s); got != "(unnamed)" {
			t.Errorf("got %q, want %q", got, "(unnamed)")
		}
	})

	t.Run("no events returns unnamed", func(t *testing.T) {
		s := &Session{}
		if got := fallbackName(s); got != "(unnamed)" {
			t.Errorf("got %q, want %q", got, "(unnamed)")
		}
	})
}
