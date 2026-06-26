---
id: command-query-separation
name: Command-query separation
description: A command mutates state and returns only error; a query is side-effect-free and returns a value; never return a status or bool code meant to be branched on - signal failure with error and compare it with errors.Is/As.
tags:
    - go
    - style
linters:
    - revive
---

# Command-query separation

> Split commands from queries. A command changes state and returns only `error` (or `(T, error)` when
> it must hand back what it created). A query computes a value and causes no observable side effect.
> Never return a status integer or bool that the caller branches on to discover success - that is what
> `error` plus `errors.Is`/`errors.As` is for.

Applies to: designing a function's return contract. Reinforces [[errors]] (return `(T, error)` last,
sentinels compared by identity) and [[effective-go]].

{{define "when"}}
- Deciding what a state-changing method returns.
- Writing a function the caller branches on for success/failure.
- Reviewing a query that quietly mutates, or a command that returns a status code.
{{end}}

{{define "must"}}
1. A command (it mutates state) returns only `error`, or `(T, error)` when the caller needs the
   created/updated value back. Failure is an `error`, never a sentinel int/bool status.
2. A query (it returns a value) is side-effect-free: calling it twice changes nothing observable; it
   does not mutate the receiver, write to a store, or emit an event.
3. Callers branch on failure via `errors.Is`/`errors.As` against a sentinel or the domain-error type
   (see [[errors]]), never on a returned status code or a magic return value (`-1`, `0`, `""`).
4. Do not blend the two: a method that both mutates AND returns a queried value the caller reads for
   control flow should be split into a command and a query.
{{end}}

{{define "forbidden"}}
- Returning a status/result code or a bool the caller branches on instead of an `error`.
- A query with an observable side effect (mutation, write, event).
- A magic return sentinel (`-1`/`0`/`""`) standing in for an error.
- One method that mutates state and returns a value consumed for control flow.
{{end}}

{{define "validation"}}
- [ ] Commands return only `error` (or `(T, error)`); no status/bool success codes.
- [ ] Queries are side-effect-free and idempotent to call.
- [ ] Callers branch via `errors.Is`/`errors.As`, not on a status code or magic value.
- [ ] No single method mixes a state mutation with a control-flow query result.
{{end}}
