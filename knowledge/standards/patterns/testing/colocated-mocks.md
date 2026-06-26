---
id: colocated-mocks
name: Colocated mocks
description: Mocks live in the same package as the interface, in <source>_mock_test.go; generated with mockgen; hand-rolled only for <= 2 methods; dumb data + bare returns.
tags:
    - go
    - testing
dependencies:
    - testing
---

# Colocated mocks

> Mocks live in the same Go package as the interface they implement, in `<source>_mock_test.go`.
> The `_test.go` suffix makes them test-only. One mock file per source file. Generated with
> `mockgen`; hand-rolled only for <= 2 methods. Mocks are dumb data + bare returns - no logic.

Reference tools: `go.uber.org/mock` (mockgen).

{{define "when"}}
- Mocking a dependency for an isolated unit test.
- Adding a `//go:generate mockgen` directive.
{{end}}

{{define "recipe"}}
File naming:

| Source                                        | Mock                      |
|-----------------------------------------------|---------------------------|
| `dispatcher.go` (declares `Dispatcher`)       | `dispatcher_mock_test.go` |
| `interfaces.go` (declares `Writer`, `Reader`) | `interfaces_mock_test.go` |

Generated mock (the `//go:generate` directive goes AFTER the import block of an existing source file,
never before `package`):

```go
//go:generate go tool -modfile=tools/go.mod mockgen \
//   -source=dispatcher.go \
//   -destination=dispatcher_mock_test.go \
//   -package=notifier
```

A fake used by MULTIPLE test packages is not copied into each: it lives in a shared `<pkg>test` helper
package (e.g. `webwraptest.NewReader(ctx).WithQuery(k, v).WithBody(x)`) that the tests import. The
colocated `_mock_test.go` rule still applies to behavioral dependency mocks (>2 methods -> `mockgen`).

```bash
task tools:mocks
```

Hand-rolled mock for <= 2 methods (record every call; bare struct-field returns, no channels):

```go
// dispatcher_mock_test.go
package notifier

import "context"

type dispatcherMock struct {
    sendCalls    []struct{ channelID, content string }
    sendMessageID string
    sendErr       error
}

func (m *dispatcherMock) Send(ctx context.Context, channelID, content string) (string, error) {
    m.sendCalls = append(m.sendCalls, struct{ channelID, content string }{channelID, content})
    return m.sendMessageID, m.sendErr
}
```

Use directly (package-local, no setter ceremony):

```go
func TestNotify(t *testing.T) {
    disp := &dispatcherMock{sendMessageID: "msg-1"}
    n := New(readerMock{...}, disp)
    if err := n.NotifyPending(context.Background()); err != nil { t.Fatal(err) }
    if len(disp.sendCalls) != 1 { t.Fatalf("expected 1 Send, got %d", len(disp.sendCalls)) }
}
```
{{end}}

{{define "forbidden"}}
- A separate `mocks/` package.
- Mocks under a `testing/` build tag (use the `_test.go` suffix instead).
- Mocks importing the production code's third-party deps. Mocks are dumb data + bare returns.
- Mocks that contain logic. If your mock needs an `if`, your test is testing the mock.
{{end}}

{{define "validation"}}
- [ ] Mocks colocated as `<source>_mock_test.go` in the interface's package.
- [ ] One mock file per source file; `mockgen` directive present (or hand-rolled for <= 2 methods).
- [ ] Mocks are dumb data + bare returns; no logic; no production third-party deps.
{{end}}
