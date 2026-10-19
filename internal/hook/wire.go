package hook

import (
	"encoding/json"
	"fmt"
)

type hookOutput struct {
	HookSpecificOutput hookSpecific `json:"hookSpecificOutput"`
}

type hookSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

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

// simplification: a malformed payload degrades to the zero value (handlers
// already treat that as "nothing to do") instead of a decode error. Upgrade
// path: return the error too if a handler needs "malformed" vs "empty".
func decodeHookPayload(payload []byte) hookPayload {
	var in hookPayload
	// The JSON shape must match knowledge/templates/hooks/opencode-plugin.js runHook payload.
	_ = json.Unmarshal(payload, &in)
	return in
}

func (disp *Dispatcher) emitOutput(event, context string) {
	out, err := json.Marshal(hookOutput{HookSpecificOutput: hookSpecific{
		HookEventName:     event,
		AdditionalContext: context,
	}})
	if err != nil {
		_, _ = fmt.Fprintln(disp.errOut, "hook: marshal:", err)
		return
	}
	_, _ = fmt.Fprintln(disp.out, string(out))
}
