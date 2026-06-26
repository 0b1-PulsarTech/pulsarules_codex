---
id: observer-weakptr
name: Observer with weak.Pointer
description: A weak.Pointer observer registry lets the GC drop observers when their owner releases it; sweeps stale entries on Notify; long-lived observers hold a strong reference.
tags:
    - go
    - concurrency
dependencies:
    - concurrency
---

# Observer with weak.Pointer

> A strong-reference observer slice keeps every observer alive forever; `weak.Pointer[Observer]`
> lets the GC drop an observer when its owner releases it; the registry sweeps stale entries on the
> next `Notify`. Long-lived observers must hold a strong reference at the call site (e.g. registered
> as an injector instance).

Reference tools: Go 1.24+ `weak` package.

{{define "when"}}
- An in-process observer/registry where observers may come and go.
- Ephemeral fan-out where explicit `Unregister` would leak.
{{end}}

{{define "recipe"}}
```go
import "weak"

type Observer interface {
    OnEvent(ctx context.Context, e Event)
}

type observerSet struct {
    mu    sync.Mutex
    items []weak.Pointer[Observer]
}

func (s *observerSet) Register(o *Observer) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.items = append(s.items, weak.Make(o))
}

func (s *observerSet) Notify(ctx context.Context, e Event) {
    s.mu.Lock()
    defer s.mu.Unlock()
    live := s.items[:0]
    for _, w := range s.items {
        p := w.Value()
        if p == nil {
            continue // GC'd - drop silently
        }
        (*p).OnEvent(ctx, e)
        live = append(live, w)
    }
    s.items = live
}
```

Keep a long-lived observer alive (strong ref in the injector):

```go
var obs notifier.Observer = discordSync
notifier.RegisterObserver(&obs)
remy.RegisterInstance(inj, &obs, "discordsync:observer")
```
{{end}}

{{define "forbidden"}}
- A strong-reference observer slice with no `Unregister` (leak).
- Registering the same observer twice (the sweep doesn't deduplicate).
- Passing a non-addressable interface pointer to `Register`.
{{end}}

{{define "validation"}}
- [ ] Registry uses `weak.Pointer[Observer]`; sweeps stale entries on `Notify`.
- [ ] Long-lived observers hold a strong reference (e.g. injector instance).
- [ ] Go 1.24+; interface pointer passed to `Register` is addressable.
{{end}}
