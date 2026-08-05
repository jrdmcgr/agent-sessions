package main

import (
	"testing"
	"time"
)

func TestHumanizeTokens(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1k"},
		{1500, "2k"},
		{2500, "2k"},
		{999_499, "999k"},
		{1_000_000, "1.0M"},
		{12_345_678, "12.3M"},
		{1_500_000_000, "1.5B"},
	}
	for _, c := range cases {
		if got := humanizeTokens(c.n); got != c.want {
			t.Errorf("humanizeTokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestHumanizeDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0m"},
		{59*time.Minute + 59*time.Second, "59m"},
		{60 * time.Minute, "1h00m"},
		{3*time.Hour + 7*time.Minute, "3h07m"},
		{25*time.Hour + 5*time.Minute, "25h05m"},
	}
	for _, c := range cases {
		if got := humanizeDuration(c.d); got != c.want {
			t.Errorf("humanizeDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestHumanizeModels(t *testing.T) {
	longA := "extremely-long-model-name-a"
	longB := "extremely-long-model-name-b"

	cases := []struct {
		models []string
		want   string
	}{
		{nil, "-"},
		{[]string{"opus-5"}, "opus-5"},
		{[]string{"a", "b"}, "a,b"},
		{[]string{"a", "b", "c"}, "a,b+1"},
		{[]string{"a", "b", "c", "d"}, "a,b+2"},
	}
	for _, c := range cases {
		if got := humanizeModels(c.models); got != c.want {
			t.Errorf("humanizeModels(%v) = %q, want %q", c.models, got, c.want)
		}
	}

	// Two long names whose join exceeds 22 bytes: truncate to 21 bytes + "~".
	head := longA + "," + longB
	if len(head) <= 22 {
		t.Fatalf("test fixture invalid: head %q is not long enough (%d bytes)", head, len(head))
	}
	want := head[:21] + "~"
	if got := humanizeModels([]string{longA, longB}); got != want {
		t.Errorf("humanizeModels([longA, longB]) = %q, want %q", got, want)
	}

	// With a third model, the truncated head gets "+1" appended.
	wantWithThird := want + "+1"
	if got := humanizeModels([]string{longA, longB, "c"}); got != wantWithThird {
		t.Errorf("humanizeModels([longA, longB, c]) = %q, want %q", got, wantWithThird)
	}
}
