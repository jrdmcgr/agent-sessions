package main

import "strings"

// piToolEntry is a pi tool name's Claude spelling plus any argument-key
// renames needed to match Claude's input shape.
type piToolEntry struct {
	name   string
	keyMap map[string]string
}

// piToolMap translates pi tool names (and argument keys) to the Claude
// spellings the rest of the pipeline and the note renderer expect. Ported
// from archive-session's _PI_TOOL_MAP. Lookup is case-insensitive; an unknown
// tool keeps its original name and arguments unchanged.
var piToolMap = map[string]piToolEntry{
	"bash":  {name: "Bash"},
	"read":  {name: "Read", keyMap: map[string]string{"path": "file_path"}},
	"write": {name: "Write", keyMap: map[string]string{"path": "file_path"}},
	"edit":  {name: "Edit", keyMap: map[string]string{"path": "file_path"}},
	"glob":  {name: "Glob"},
	"grep":  {name: "Grep"},
}

// piContentBlocks translates pi message content into normalized Blocks. A bare
// string becomes a single text block. text -> text, toolCall -> tool_use with
// the name/arg remap; thinking (and anything else) is dropped, matching
// archive-session's _pi_content_blocks.
func piContentBlocks(content any) []Block {
	if s, ok := content.(string); ok {
		return []Block{{Type: "text", Text: s}}
	}
	list, ok := content.([]any)
	if !ok {
		return nil
	}
	var out []Block
	for _, item := range list {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch getString(block, "type") {
		case "text":
			out = append(out, Block{Type: "text", Text: getString(block, "text")})
		case "toolCall":
			rawName := getString(block, "name")
			if rawName == "" {
				rawName = "tool"
			}
			name := rawName
			var keyMap map[string]string
			if entry, found := piToolMap[strings.ToLower(rawName)]; found {
				name = entry.name
				keyMap = entry.keyMap
			}
			input := map[string]any{}
			if args := getMap(block, "arguments"); args != nil {
				for k, v := range args {
					if dst, renamed := keyMap[k]; renamed {
						input[dst] = v
					} else {
						input[k] = v
					}
				}
			}
			out = append(out, Block{Type: "tool_use", Name: name, Input: input})
		}
	}
	return out
}

// claudeContentBlocks normalizes Claude message content, which is already in
// Anthropic block shape. A bare string becomes one text block. text and
// tool_use pass through; thinking, tool_result, and anything else are dropped,
// matching archive-session's render_assistant.
func claudeContentBlocks(content any) []Block {
	if s, ok := content.(string); ok {
		return []Block{{Type: "text", Text: s}}
	}
	list, ok := content.([]any)
	if !ok {
		return nil
	}
	var out []Block
	for _, item := range list {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch getString(block, "type") {
		case "text":
			out = append(out, Block{Type: "text", Text: getString(block, "text")})
		case "tool_use":
			out = append(out, Block{Type: "tool_use", Name: getString(block, "name"), Input: getMap(block, "input")})
		}
	}
	return out
}
