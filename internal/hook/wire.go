package hook

import (
	"encoding/json"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/applypatch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/dialect"
)

type hookPayload struct {
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd"`
	ToolName       string `json:"tool_name"`
	StopHookActive bool   `json:"stop_hook_active"`
	ToolInput      struct {
		FilePath string `json:"file_path"`
		Pattern  string `json:"pattern"`
		Glob     string `json:"glob"`
		Path     string `json:"path"`
		Command  string `json:"command"`
	} `json:"tool_input"`
	// patchOp is how a Codex patch touches ToolInput.FilePath; the zero value means "not a patch".
	patchOp applypatch.Op
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

func decodeHookPayloads(payload []byte, dialectName string) []hookPayload {
	in := decodeHookPayload(payload)
	if dialectName != dialect.Codex || in.ToolInput.FilePath != "" ||
		!isCodexEditTool(in.ToolName) {
		return []hookPayload{in}
	}
	// Codex reports an edit as patch text in tool_input.command: fan a multi-file patch out.
	edits := applypatch.Edits(in.ToolInput.Command, in.CWD)
	if len(edits) == 0 {
		return []hookPayload{in}
	}
	payloads := make([]hookPayload, 0, len(edits))
	for _, patched := range edits {
		edit := in
		edit.ToolInput.FilePath = patched.Path
		edit.patchOp = patched.Op
		payloads = append(payloads, edit)
	}
	return payloads
}

func isCodexEditTool(name string) bool {
	switch name {
	case "apply_patch", "Edit", "Write", "MultiEdit":
		return true
	default:
		return false
	}
}

func (disp *Dispatcher) emitOutput(event, context string) {
	disp.output.Add(event, context)
}

func (disp *Dispatcher) flushOutput() {
	line, err := disp.output.Take()
	if err != nil {
		_, _ = disp.errOut.Write([]byte("hook: " + err.Error() + "\n"))
		return
	}
	_, _ = disp.out.Write(line)
}
