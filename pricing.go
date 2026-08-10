package main

import "strings"

// rates are USD per million tokens.
type rates struct {
	Input, Output, CacheWrite, CacheRead float64
}

// USD per million tokens, keyed by bare model name (post-shortModel, minus
// the "claude-" prefix restored). Regenerate with `bin/update-pricing`,
// which pulls rates from LiteLLM's model_prices_and_context_window.json. Do
// not hand-edit the generated block; add new model names to the MODELS list
// in bin/update-pricing instead.
// pricing:generated:begin
var pricing = map[string]rates{
	"claude-opus-5":     {5, 25, 6.25, 0.5},
	"claude-opus-4-8":   {5, 25, 6.25, 0.5},
	"claude-opus-4-7":   {5, 25, 6.25, 0.5},
	"claude-opus-4-1":   {15, 75, 18.75, 1.5},
	"claude-opus-4":     {15, 75, 18.75, 1.5},
	"claude-sonnet-4-5": {3, 15, 3.75, 0.3},
	"claude-sonnet-4":   {3, 15, 3.75, 0.3},
	"claude-haiku-4-5":  {1, 5, 1.25, 0.1},
	"claude-3-5-haiku":  {0.8, 4, 1, 0.08},
}

// pricing:generated:end

// Claude Code writes bare aliases when a session is started with one.
var modelAliases = map[string]string{
	"opus":   "claude-opus-5",
	"sonnet": "claude-sonnet-4-5",
	"haiku":  "claude-haiku-4-5",
}

// shortModel ports short_model: "" -> ""; resolve alias; take the part after
// the last "/"; strip a leading "claude-" prefix.
func shortModel(model string) string {
	if model == "" {
		return ""
	}
	if resolved, ok := modelAliases[model]; ok {
		model = resolved
	}
	if idx := strings.LastIndex(model, "/"); idx != -1 {
		model = model[idx+1:]
	}
	return strings.TrimPrefix(model, "claude-")
}

// price ports price: cost in USD for usage under model's rates, or nil if
// the model (after alias resolution) isn't in the pricing table.
func price(model string, u Usage) *float64 {
	resolved := model
	if r, ok := modelAliases[model]; ok {
		resolved = r
	}
	r, ok := pricing[resolved]
	if !ok {
		return nil
	}
	cost := (float64(u.Input)*r.Input +
		float64(u.Output)*r.Output +
		float64(u.CacheWrite)*r.CacheWrite +
		float64(u.CacheRead)*r.CacheRead) / 1_000_000
	return &cost
}
