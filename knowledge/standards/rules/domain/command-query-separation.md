---
id: command-query-separation
name: Command-query separation
description: A command mutates state and returns only an error outcome; a query is side-effect-free and returns a value; never return a status or bool code meant to be branched on - signal failure with an error value compared by identity, not by string or status code.
tags:
    - style
---

# Command-query separation

> Split commands from queries. A command changes state and returns only an error outcome (plus the
> created/updated value when the caller needs it back). A query computes a value and causes no
> observable side effect. Never return a status integer or bool that the caller branches on to
> discover success - an error value, compared by identity/type, is for that.

Applies to: designing a function's/method's return contract, in any language. In a Go project this
reinforces [[errors]] (return `(T, error)` last, sentinels compared by identity via `errors.Is`/
`errors.As`) and [[effective-go]].

{{define "when"}}
- Deciding what a state-changing method returns.
- Writing a function the caller branches on for success/failure.
- Reviewing a query that quietly mutates, or a command that returns a status code.
{{end}}

{{define "must"}}
1. A command (it mutates state) returns only an error outcome, or the error plus the created/updated
   value when the caller needs it back. Failure is an error value, never a sentinel int/bool status.
2. A query (it returns a value) is side-effect-free: calling it twice changes nothing observable; it
   does not mutate the receiver, write to a store, or emit an event.
3. Callers branch on failure by comparing the error's identity or type against a sentinel or the
   domain-error contract (see [[errors]]), never on a returned status code or a magic return value
   (`-1`, `0`, `""`).
4. Do not blend the two: a method that both mutates and returns a queried value the caller reads for
   control flow should be split into a command and a query.
{{end}}

{{define "forbidden"}}
- Returning a status/result code or a bool the caller branches on instead of an error value.
- A query with an observable side effect (mutation, write, event).
- A magic return sentinel (`-1`/`0`/`""`) standing in for an error.
- One method that mutates state and returns a value consumed for control flow.
{{end}}

{{define "validation"}}
- [ ] Commands return only an error outcome (or the value plus the error); no status/bool codes.
- [ ] Queries are side-effect-free and idempotent to call.
- [ ] Callers branch on the error's identity/type, not on a status code or magic value.
- [ ] No single method mixes a state mutation with a control-flow query result.
{{end}}
