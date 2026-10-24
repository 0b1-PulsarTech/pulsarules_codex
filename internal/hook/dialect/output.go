package dialect

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const (
	Claude = "claude"
	Codex  = "codex"
)

type hookOutput struct {
	HookSpecificOutput hookSpecific `json:"hookSpecificOutput"`
}

type hookSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

type stopBlock struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type Output struct {
	dialect  string
	event    string
	contexts []string
}

func NewOutput(dialect string) *Output {
	return &Output{dialect: dialect}
}

func (out *Output) Add(event, context string) {
	if out.event == "" {
		out.event = event
	}
	if !slices.Contains(out.contexts, context) {
		out.contexts = append(out.contexts, context)
	}
}

func (out *Output) Take() ([]byte, error) {
	event, contexts := out.event, out.contexts
	out.event, out.contexts = "", nil
	if len(contexts) == 0 {
		return nil, nil
	}
	text := strings.Join(contexts, "\n\n")
	var value any = hookOutput{HookSpecificOutput: hookSpecific{
		HookEventName: event, AdditionalContext: text,
	}}
	if out.dialect == Codex && event == "Stop" {
		value = stopBlock{Decision: "block", Reason: text}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal hook output: %w", err)
	}
	return append(encoded, '\n'), nil
}
