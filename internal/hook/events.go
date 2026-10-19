package hook

type handler func(disp *Dispatcher, session *sessionTracker, in hookPayload) (stopOutcome, error)

type Event struct {
	Host    string
	Matcher string
	Mode    string
	Timeout int
	handle  handler
}

func plain(run func(*Dispatcher, *sessionTracker, hookPayload) error) handler {
	return func(disp *Dispatcher, session *sessionTracker, in hookPayload) (stopOutcome, error) {
		return stopOutcome{}, run(disp, session, in)
	}
}

func Events() []Event {
	const stopTimeoutSeconds = 90

	return []Event{
		{
			Host: "SessionStart", Mode: "session-start",
			handle: plain(func(disp *Dispatcher, session *sessionTracker, _ hookPayload) error {
				return disp.emitSessionStart(session)
			}),
		},
		{
			Host: "PreToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "pre-edit",
			handle: plain((*Dispatcher).emitPreEdit),
		},
		{
			Host: "PreToolUse", Matcher: "Grep|Glob|Bash", Mode: "pre-search",
			handle: plain((*Dispatcher).emitPreSearch),
		},
		{
			Host: "PostToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "post-edit",
			handle: plain((*Dispatcher).emitPostEdit),
		},
		{
			Host: "UserPromptSubmit", Mode: "user-prompt",
			handle: plain(func(disp *Dispatcher, _ *sessionTracker, _ hookPayload) error {
				return disp.emitUserPrompt()
			}),
		},
		{
			Host: "SubagentStart", Mode: "subagent-start",
			handle: plain(func(disp *Dispatcher, _ *sessionTracker, _ hookPayload) error {
				return disp.emitSubagentStart()
			}),
		},
		// The host default is 600s and it discards output past the timeout; the Stop
		// deadline must stay under this by the margin the host/claude test pins.
		{
			Host: "Stop", Mode: "stop", Timeout: stopTimeoutSeconds, handle: stopHandler,
		},
		{
			Host: "SessionEnd", Mode: "session-end",
			handle: plain(func(_ *Dispatcher, session *sessionTracker, _ hookPayload) error {
				session.Cleanup()
				return nil
			}),
		},
	}
}

func stopHandler(disp *Dispatcher, session *sessionTracker, _ hookPayload) (stopOutcome, error) {
	return disp.emitStop("Stop", session)
}
