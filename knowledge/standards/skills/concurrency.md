---
id: concurrency
name: Concurrency
---

Concurrency governs goroutine ownership, context propagation, and safe use of channels and
mutexes. Reach for it whenever you spawn a goroutine or fan out work, use sync.Mutex,
sync.WaitGroup, or channels, design a long-running worker or relay loop, or need to propagate or
honor context cancellation. It pairs with the observer-weakptr pattern for ephemeral fan-out and
dispatch-pool for worker pools. A change under this skill is not done until go test -race passes
- the race detector is never silenced.

The rules below are the composed concurrency rule.
