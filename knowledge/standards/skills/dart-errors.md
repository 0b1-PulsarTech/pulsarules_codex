---
id: dart-errors
name: Dart errors
---

Governs how a Dart function delivers failure: a result record returned by the function, either
positional with the error last, `(T?, DomainError?)`, or named, `({T? value, DomainError? error})`,
never a thrown error, because Dart carries no `throws` clause and a thrown error is invisible at
both the call site and the compiler. Reach for it whenever writing a Dart function that can fail
with a domain error, deciding throw-vs-return, or consuming a result record at a call site. It is
the direct Dart analogue of this repo's `errors-logging`: the same "failure is part of the return
type" principle Go states as `(T, error)`, delivered through Dart 3's record type instead. It does
not redefine the error type - the `sealed DomainError` hierarchy stays exactly what `dart-style`
already specifies; this skill governs only how that value reaches the caller.

The rules below are the composed dart-errors rule.
