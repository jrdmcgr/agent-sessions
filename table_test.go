package main

import (
	"bytes"
	"testing"
	"time"
)

func mkTestRows() []Row {
	base := time.Date(2024, 8, 5, 9, 0, 0, 0, time.Local)
	return []Row{
		{
			Date:    dayOf(base),
			Project: "agent-sessions",
			Name:    "fix-widths",
			Active:  true,
			Start:   base,
			End:     base.Add(1*time.Hour + 23*time.Minute),
			Harness: "pi",
			Models:  []string{"sonnet-4-5"},
			Tokens:  12345,
			Cost:    1.2345,
			Priced:  true,
		},
		{
			Date:    dayOf(base),
			Project: "dotfiles",
			Name:    "a-much-longer-session-name",
			Active:  false,
			Start:   base.Add(2 * time.Hour),
			End:     base.Add(2*time.Hour + 5*time.Minute),
			Harness: "claude",
			Models:  []string{"gpt-5-mystery"},
			Tokens:  500,
			Cost:    0.5,
			Priced:  false,
		},
	}
}

func TestRenderTableEmpty(t *testing.T) {
	var w, errW bytes.Buffer
	renderTable(&w, &errW, nil, false)
	want := "No sessions found.\n"
	if w.String() != want {
		t.Errorf("stdout = %q, want %q", w.String(), want)
	}
	if errW.String() != "" {
		t.Errorf("errW = %q, want empty", errW.String())
	}
}

func TestRenderTableWithDateUnpriced(t *testing.T) {
	rows := mkTestRows()
	var w, errW bytes.Buffer
	renderTable(&w, &errW, rows, true)

	want := "" +
		"╭───────────┬────────────────┬────────────────────────────┬─────────────┬───────┬─────────┬───────────────┬────────┬────────╮\n" +
		"│ DATE      │ PROJECT        │ SESSION                    │ TIME        │ DUR   │ HARNESS │ MODEL         │ TOKENS │ COST   │\n" +
		"├───────────┼────────────────┼────────────────────────────┼─────────────┼───────┼─────────┼───────────────┼────────┼────────┤\n" +
		"│ Mon 08-05 │ agent-sessions │ fix-widths *               │ 09:00-10:23 │ 1h23m │ pi      │ sonnet-4-5    │ 12k    │ $1.23  │\n" +
		"│ Mon 08-05 │ dotfiles       │ a-much-longer-session-name │ 11:00-11:05 │ 5m    │ claude  │ gpt-5-mystery │ 500    │ $0.50? │\n" +
		"│ TOTAL     │                │ 2 session-days             │             │ 1h28m │         │               │ 13k    │ $1.73? │\n" +
		"╰───────────┴────────────────┴────────────────────────────┴─────────────┴───────┴─────────┴───────────────┴────────┴────────╯\n"

	if w.String() != want {
		t.Errorf("stdout mismatch\ngot:\n%s\nwant:\n%s", w.String(), want)
	}

	wantErr := "\n? = includes an unpriced model; cost is a lower bound.\n"
	if errW.String() != wantErr {
		t.Errorf("errW = %q, want %q", errW.String(), wantErr)
	}
}

func TestRenderTableNoDateAllPriced(t *testing.T) {
	rows := mkTestRows()
	rows[1].Priced = true // make all rows priced

	var w, errW bytes.Buffer
	renderTable(&w, &errW, rows, false)

	want := "" +
		"╭────────────────┬────────────────────────────┬─────────────┬───────┬─────────┬───────────────┬────────┬───────╮\n" +
		"│ PROJECT        │ SESSION                    │ TIME        │ DUR   │ HARNESS │ MODEL         │ TOKENS │ COST  │\n" +
		"├────────────────┼────────────────────────────┼─────────────┼───────┼─────────┼───────────────┼────────┼───────┤\n" +
		"│ agent-sessions │ fix-widths *               │ 09:00-10:23 │ 1h23m │ pi      │ sonnet-4-5    │ 12k    │ $1.23 │\n" +
		"│ dotfiles       │ a-much-longer-session-name │ 11:00-11:05 │ 5m    │ claude  │ gpt-5-mystery │ 500    │ $0.50 │\n" +
		"│ TOTAL          │ 2 sessions                 │             │ 1h28m │         │               │ 13k    │ $1.73 │\n" +
		"╰────────────────┴────────────────────────────┴─────────────┴───────┴─────────┴───────────────┴────────┴───────╯\n"

	if w.String() != want {
		t.Errorf("stdout mismatch\ngot:\n%s\nwant:\n%s", w.String(), want)
	}
	if errW.String() != "" {
		t.Errorf("errW = %q, want empty", errW.String())
	}
}
