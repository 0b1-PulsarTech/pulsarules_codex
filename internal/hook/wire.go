package hook

import (
	"encoding/json"
	"fmt"
	"os"
)

// Output is the non-blocking JSON Claude Code reads from a hook's stdout.
type Output struct {
	HookSpecificOutput Specific `json:"hookSpecificOutput"`
}

// Specific carries the event name and additional context for the agent.
type Specific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// hookPayload is decoded once per Dispatch call so handlers share one parse
// of the stdin bytes instead of each re-unmarshalling them.
type hookPayload struct {
	SessionID string `json:"session_id"`
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
		Pattern  string `json:"pattern"`
		Glob     string `json:"glob"`
		Path     string `json:"path"`
		Command  string `json:"command"`
	} `json:"tool_input"`
}

// simplification: a malformed payload degrades to the zero value (every field
// empty) instead of surfacing a decode error, because a hook must never block
// the agent's turn over a parse failure; every handler already treats its
// fields' zero values as "nothing to do".
func decodeHookPayload(payload []byte) hookPayload {
	var in hookPayload
	_ = json.Unmarshal(payload, &in)
	return in
}

func (d *Dispatcher) emitOutput(event, context string) {
	out, err := json.Marshal(Output{HookSpecificOutput: Specific{
		HookEventName:     event,
		AdditionalContext: context,
	}})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "hook: marshal:", err)
		return
	}
	_, _ = fmt.Fprintln(d.out, string(out))
}
