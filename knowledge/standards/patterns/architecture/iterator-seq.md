---
id: iterator-seq
name: Iterator with iter.Seq
description: Expose a traversal as a Go 1.23+ iter.Seq / iter.Seq2 push iterator - validate eagerly and return the error, then a func(yield) that stops when yield returns false; range-over-func lets callers break, filter, and compose without materializing a slice.
tags:
    - go
    - architecture
dependencies:
    - effective-go
---

# Iterator with iter.Seq

> When a producer yields a sequence a caller consumes lazily, return an `iter.Seq[T]`
> (`func(yield func(T) bool)`) instead of a materialized `[]T` or a hand-rolled cursor. Validate the
> inputs EAGERLY and return `(iter.Seq[T], error)` so setup errors surface before iteration; the
> sequence body then yields items and MUST stop when `yield` returns false so `break`/early-return in
> the caller's `range` actually halts work. Use `iter.Seq2[K,V]` for pairs.

Reference: `terectek_comms` `libs/tereckernel/permitek/internal/permwire` (`Decode` returns
`(iter.Seq[Entry], error)`, validating the header eagerly then yielding entries).

{{define "when"}}
- A producer yields many items a caller consumes one at a time (decode, scan, paginate, walk a tree).
- The caller wants to break early, filter, or compose the traversal without a full slice in memory.
- A traversal currently returns a big `[]T`, exposes an ad-hoc cursor, or takes a visitor callback.
{{end}}

{{define "recipe"}}
```go
// Validate eagerly, return the error; then hand back a lazy sequence.
func Decode(encoded []byte) (iter.Seq[Entry], error) {
    if len(encoded) == 0 {
        return nil, fmt.Errorf("%w: empty", ErrMalformed)
    }
    body := encoded[1:]

    return func(yield func(Entry) bool) {
        for off := 0; off < len(body); off += entrySize {
            if !yield(entryAt(body, off)) {
                return // caller broke - stop, run no more work
            }
        }
    }, nil
}
```

Consume it with range-over-func; `break` propagates through `yield` returning false:

```go
for entry := range seq {
    if entry.ID == want {
        found = entry
        break
    }
}
```

For key/value traversal use `iter.Seq2[K, V]` (`func(yield func(K, V) bool)`). If the sequence owns a
resource (a file, a DB rows handle), open it inside the function and `defer close` there so it is
released when iteration ends or the caller breaks.
{{end}}

{{define "forbidden"}}
- A sequence body that ignores `yield`'s bool return - the caller cannot break and work runs on.
- Returning `iter.Seq[T]` but doing the validation lazily inside the closure, so a setup error only
  surfaces mid-range instead of at the call.
- Materializing a large `[]T` just to `range` it once when the caller consumes lazily.
- Leaking a resource opened for the iteration when the caller breaks early (open and defer-close it
  inside the sequence function).
{{end}}

{{define "validation"}}
- [ ] Traversal returns `iter.Seq[T]` / `iter.Seq2[K,V]`, not a big slice or ad-hoc cursor.
- [ ] Inputs validated eagerly; the error is returned alongside the sequence, before iteration.
- [ ] The body stops when `yield` returns false (early break is honored).
- [ ] Any resource the sequence owns is closed when iteration ends or the caller breaks.
{{end}}
