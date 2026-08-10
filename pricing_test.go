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
		want := 2.0 + 10.0 + 2.5 + 0.2
		if *got != want {
			t.Errorf("price = %v, want %v", *got, want)
		}
	})

	t.Run("dated snapshot falls back to bare model rates", func(t *testing.T) {
		dated := price("claude-haiku-4-5-20251001", Usage{Input: 1_000_000})
		bare := price("claude-haiku-4-5", Usage{Input: 1_000_000})
		if dated == nil || bare == nil {
			t.Fatal("price returned nil")
		}
		if *dated != *bare {
			t.Errorf("dated price = %v, want %v (bare model rate)", *dated, *bare)
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
}
