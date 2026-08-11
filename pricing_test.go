package main

import "testing"

func TestShortModel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"opus", "opus-5"},
		{"claude-sonnet-4-5", "sonnet-4-5"},
		{"anthropic/claude-opus-4-1", "opus-4-1"},
		{"gpt-5", "gpt-5"},
	}
	for _, c := range cases {
		if got := shortModel(c.in); got != c.want {
			t.Errorf("shortModel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPrice(t *testing.T) {
	t.Run("known model input only", func(t *testing.T) {
		got := price("claude-sonnet-4-5", Usage{Input: 1_000_000})
		if got == nil {
			t.Fatal("price returned nil for known model")
		}
		if *got != 3.0 {
			t.Errorf("price = %v, want 3.0", *got)
		}
	})

	t.Run("known model all categories", func(t *testing.T) {
		u := Usage{Input: 1e6, Output: 1e6, CacheWrite: 1e6, CacheRead: 1e6}
		got := price("claude-sonnet-4-5", u)
		if got == nil {
			t.Fatal("price returned nil for known model")
		}
		want := 3.0 + 15.0 + 3.75 + 0.3
		if *got != want {
			t.Errorf("price = %v, want %v", *got, want)
		}
	})

	t.Run("alias matches resolved model", func(t *testing.T) {
		u := Usage{Input: 1e6, Output: 1e6, CacheWrite: 1e6, CacheRead: 1e6}
		got := price("sonnet", u)
		if got == nil {
			t.Fatal("price returned nil for alias")
		}
		want := 3.0 + 15.0 + 3.75 + 0.3
		if *got != want {
			t.Errorf("price = %v, want %v", *got, want)
		}
	})

	t.Run("unknown model", func(t *testing.T) {
		got := price("gpt-5", Usage{Input: 1_000_000})
		if got != nil {
			t.Errorf("price = %v, want nil", *got)
		}
	})

	t.Run("zero usage known model", func(t *testing.T) {
		got := price("claude-sonnet-4-5", Usage{})
		if got == nil {
			t.Fatal("price returned nil for known model with zero usage")
		}
		if *got != 0.0 {
			t.Errorf("price = %v, want 0.0", *got)
		}
	})

	// Regression: catches CacheWrite/CacheRead swapped in the rates table or
	// in the price formula. CacheWrite and CacheRead have different rates
	// for sonnet-4-5 (3.75 vs 0.3), so swapping them changes the result.
	t.Run("cache write and cache read distinguished", func(t *testing.T) {
		write := price("claude-sonnet-4-5", Usage{CacheWrite: 1_000_000})
		read := price("claude-sonnet-4-5", Usage{CacheRead: 1_000_000})
		if write == nil || read == nil {
			t.Fatal("price returned nil for known model")
		}
		if *write != 3.75 {
			t.Errorf("cache write price = %v, want 3.75", *write)
		}
		if *read != 0.3 {
			t.Errorf("cache read price = %v, want 0.3", *read)
		}
	})

	// Regression (task 12 parity, 2026-08-11): pricing.go had drifted from
	// the Python spec three ways: opus rates were halved/wrong, "sonnet"
	// aliased to a bare "claude-sonnet-5" entry the Python table has never
	// had, and price() stripped a "-YYYYMMDD" snapshot suffix before giving
	// up, which Python does not do. All three must match Python exactly:
	// only 9 rows, opus at (15,75,18.75,1.5), "sonnet" -> sonnet-4-5, and
	// dated snapshots unpriced.
	t.Run("opus rate matches python (not the halved litellm-synced value)", func(t *testing.T) {
		got := price("claude-opus-5", Usage{Input: 1_000_000})
		if got == nil {
			t.Fatal("price returned nil for claude-opus-5")
		}
		if *got != 15.0 {
			t.Errorf("price(claude-opus-5, 1M input) = %v, want 15.0", *got)
		}
	})

	t.Run("sonnet alias resolves to sonnet-4-5, not a bare sonnet-5 entry", func(t *testing.T) {
		got := price("sonnet", Usage{Input: 1_000_000})
		want := price("claude-sonnet-4-5", Usage{Input: 1_000_000})
		if got == nil || want == nil || *got != *want {
			t.Errorf("price(sonnet) = %v, want price(claude-sonnet-4-5) = %v", got, want)
		}
		// claude-sonnet-5 as a literal (unaliased) model string must stay
		// unpriced: Python's PRICING table has no such key.
		if p := price("claude-sonnet-5", Usage{Input: 1_000_000}); p != nil {
			t.Errorf("price(claude-sonnet-5) = %v, want nil (not in python's table)", *p)
		}
	})

	t.Run("dated snapshot suffix is not stripped", func(t *testing.T) {
		if p := price("claude-haiku-4-5-20251001", Usage{Input: 1_000_000}); p != nil {
			t.Errorf("price(claude-haiku-4-5-20251001) = %v, want nil (python has no snapshot fallback)", *p)
		}
	})
}
