---
id: verification-go
name: Verification before done - Go command gate
description: The Go project's concrete definition-of-done gate - go build, go vet, go test -race, gofmt -l, and the project linter - instantiating the neutral verification-before-done contract.
tags:
    - go
    - process
---

# Verification before done - Go command gate

> A Go project's definition-of-done gate is `go build ./...`, `go vet ./...`, `go test -race ./...`,
> `gofmt -l` clean, and the project linter (`task lint` / `golangci-lint run`). This is the Go
> instantiation of [[verification]]'s neutral gate; run both together.

Applies to: finishing any change in a Go project, before reporting it done or committing.

{{define "when"}}
- About to report a task complete or write a commit in a Go project.
- Claiming a Go change is fixed, passing, or working.
{{end}}

{{define "must"}}
1. Run `go build ./...` and `go vet ./...` clean before declaring done.
2. Run `go test -race ./...`; the race detector stays on, never silenced or skipped to get a pass.
3. Run `gofmt -l` and confirm it lists no files; run the project linter (`task lint` /
   `golangci-lint run`) clean.
{{end}}

{{define "forbidden"}}
- Committing or reporting complete while `go build`, `go vet`, `go test -race`, `gofmt -l`, or the
  linter is red.
- Silencing or skipping `-race` to get a green test run.
{{end}}

{{define "validation"}}
- [ ] `go build ./...` and `go vet ./...` are clean.
- [ ] `go test -race ./...` is green with the race detector on.
- [ ] `gofmt -l` lists no files; the project linter is clean.
{{end}}
