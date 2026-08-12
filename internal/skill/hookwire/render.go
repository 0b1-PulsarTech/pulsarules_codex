package hookwire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"text/template"
)

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

type hooksBlock struct {
	Hooks map[string][]hookGroup `json:"hooks"`
}

// hookRegistration is one skill-router-reminder invocation the settings
// template renders: the settings.json event, the tool matcher scoping it
// (empty for events with none), and the mode argument passed to the
// script. Every registration renders an identical command shape (mirroring
// managedServers in mcpwire); add a hook by appending a row here.
type hookRegistration struct {
	Event   string
	Matcher string
	Mode    string
}

// hookRegistrations are the skill-router-reminder hook events wired into
// settings.json.
var hookRegistrations = []hookRegistration{
	{Event: "SessionStart", Mode: "session-start"},
	{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "pre-edit"},
	{Event: "PreToolUse", Matcher: "Grep|Glob|Bash", Mode: "pre-search"},
	{Event: "PostToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "post-edit"},
	{Event: "UserPromptSubmit", Mode: "user-prompt"},
	{Event: "SubagentStart", Mode: "subagent-start"},
	{Event: "Stop", Mode: "stop"},
	{Event: "SessionEnd", Mode: "session-end"},
}

// hookEventGroup is one settings.json event key plus its ordered
// registrations, the shape the template ranges over. JSON forbids repeating
// an object key, so two registrations sharing an Event (PreToolUse's
// pre-edit and pre-search matchers) must land in one group's Rows, not as
// two separate top-level entries that would silently overwrite each other.
type hookEventGroup struct {
	Name string
	Rows []hookRegistration
}

// groupHookRegistrations folds the flat table into one ordered group per
// distinct event, preserving first-appearance order and each event's row
// order.
func groupHookRegistrations(rows []hookRegistration) []hookEventGroup {
	groups := make([]hookEventGroup, 0, len(rows))
	index := make(map[string]int, len(rows))
	for _, row := range rows {
		i, ok := index[row.Event]
		if !ok {
			i = len(groups)
			index[row.Event] = i
			groups = append(groups, hookEventGroup{Name: row.Event})
		}
		groups[i].Rows = append(groups[i].Rows, row)
	}
	return groups
}

// RenderHooksBlock executes the settings hooks template over
// hookRegistrations and returns the indented JSON block. The hook command
// resolves the project root at runtime via $CLAUDE_PROJECT_DIR, so the block
// is portable (no per-target path baked in). It validates that the rendered
// result is well-formed JSON before it reaches a settings file.
func RenderHooksBlock(templates fs.FS) ([]byte, error) {
	tmpl, err := template.ParseFS(templates, "hooks/settings.hooks.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse hooks settings template: %w", err)
	}
	var rendered bytes.Buffer
	if err = tmpl.Execute(&rendered, groupHookRegistrations(hookRegistrations)); err != nil {
		return nil, fmt.Errorf("render hooks settings template: %w", err)
	}
	var block hooksBlock
	if err = json.Unmarshal(rendered.Bytes(), &block); err != nil {
		return nil, fmt.Errorf("render hooks block: %w", err)
	}
	out, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hooks block: %w", err)
	}
	return append(out, '\n'), nil
}
