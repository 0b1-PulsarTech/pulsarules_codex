package claude

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

type hookRegistration struct {
	Event   string
	Matcher string
	Mode    string
}

func hookRegistrations() []hookRegistration {
	return []hookRegistration{
		{Event: "SessionStart", Mode: "session-start"},
		{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "pre-edit"},
		{Event: "PreToolUse", Matcher: "Grep|Glob|Bash", Mode: "pre-search"},
		{Event: "PostToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "post-edit"},
		{Event: "UserPromptSubmit", Mode: "user-prompt"},
		{Event: "SubagentStart", Mode: "subagent-start"},
		{Event: "Stop", Mode: "stop"},
		{Event: "SessionEnd", Mode: "session-end"},
	}
}

type hookEventGroup struct {
	Name string
	Rows []hookRegistration
}

func groupHookRegistrations(rows []hookRegistration) []hookEventGroup {
	groups := make([]hookEventGroup, 0, len(rows))
	// JSON forbids a repeated key, so registrations sharing an Event land in one group.
	index := make(map[string]int, len(rows))
	for _, row := range rows {
		pos, ok := index[row.Event]
		if !ok {
			pos = len(groups)
			index[row.Event] = pos
			groups = append(groups, hookEventGroup{Name: row.Event})
		}
		groups[pos].Rows = append(groups[pos].Rows, row)
	}
	return groups
}

func RenderHooksBlock(templates fs.FS) ([]byte, error) {
	tmpl, err := template.ParseFS(templates, "hooks/settings.hooks.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse hooks settings template: %w", err)
	}
	var rendered bytes.Buffer
	if err = tmpl.Execute(&rendered, groupHookRegistrations(hookRegistrations())); err != nil {
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
