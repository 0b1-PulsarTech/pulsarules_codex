---
id: logging
name: Logging
description: Structured, typed-attribute logging obtained via constructor/context injection (never a library-global); one log per error at the chain top; no secrets/PII. Language-neutral hygiene; see logging-go for the log/slog instantiation.
tags:
    - logging
---

# Logging

> One structured logger, taking typed attributes (never a formatted/positional message), obtained by
> constructor or context injection rather than a library-global; one log per error at the chain top;
> no secrets/PII.

Applies to: all production logging, in any language or logging library.

{{define "when"}}
- Adding any log statement in production code.
- Deciding a log level or whether to log vs return.
- Choosing a logger source (global vs injected).
{{end}}

{{define "must"}}
1. Take the logger from the constructor or the request context; a library or handler never reaches
   for a global/ambient logger.
2. Attach structured, typed key-value attributes to every log call; never format variable data into
   the message string.
3. Log an error exactly once, at the top of the call chain. Do not log and return the same error
   (see [[errors]]).
4. Never log secrets, tokens, Authorization headers, raw request/response bodies, or PII. IDs are
   fine; log a redacted summary (a size or count) when unsure.
{{end}}

{{define "forbidden"}}
- A global logger inside a library instead of an injected/context logger.
- Formatting variable data into the message string instead of a structured attribute.
- Logging secrets, tokens, raw bodies, or customer PII.
- Logging an error then returning it (log or return).
{{end}}

{{define "validation"}}
- [ ] The logger is injected (constructor/context), never a library-global.
- [ ] Logging uses structured, typed attributes; no data formatted into the message string.
- [ ] Each error logged once at the chain top, never logged-and-returned.
- [ ] No secrets/tokens/PII/raw bodies in logs.
{{end}}
