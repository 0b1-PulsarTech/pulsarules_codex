---
id: dart-errors
name: Errors - Dart result record
description: A Dart function that can fail returns a result record carrying the value and the DomainError, positional with the error last or named, never throws to signal it; throwing stays for a broken invariant or a framework-mandated throw. The Dart delivery counterpart to Go's (T, error); the error type stays the sealed DomainError hierarchy from dart-style.
tags:
    - dart
    - errors
---

# Errors - Dart result record

> A Dart function whose outcome can fail returns a result record carrying both the success value and
> the error, either positional with the error last, `(T?, DomainError?)`, or named,
> `({T? value, DomainError? error})` - never throws to signal that outcome. Throwing stays for a
> broken invariant (a programmer error with no recovery) or a framework-mandated throw.

Applies to: any Dart function that can fail with a domain error - validation, a not-found lookup, an
operation the caller might retry with different input. Dart carries no `throws` clause on a function
signature, so a thrown error is invisible at the call site and to the compiler: nothing in the type
tells a caller the call can fail, and nothing forces them to handle it. This is the Dart delivery
counterpart to [[errors]]'s `(T, error)` return - the principle (failure is part of the return type,
not a side channel) is the same; only the shape differs. The error type is unchanged: the `sealed`
`DomainError` hierarchy from [[dart-style]] (rule 4) still models the closed set of failure variants -
this rule governs how that value reaches the caller, not what it is.

{{define "when"}}
- Writing a Dart function whose outcome can fail with a domain error.
- Deciding whether to throw or return a `DomainError`.
- Consuming a result record at a call site.
- Wiring a framework boundary (a `build()` override, a codegen-called factory, an `Isolate` entry
  point) that itself requires a throw.
{{end}}

{{define "must"}}
1. A function that can fail returns a result record in one of two forms, and never `throw`s to
   signal that failure: a positional record with the error last, `(T?, DomainError?)`, or a named
   record, `({T? value, DomainError? error})`. An async function returns the same record inside a
   `Future`. Both forms are accepted; pick one per package and keep it, so a caller never has to
   guess which field comes first.
2. Exactly one field is non-null on return: a null error (the `error` field of a named record, the
   last field of a positional one) means the value is the result; a non-null error means the value
   carries no meaning the caller may read. A function returning both non-null or both null is a
   defect in the function, not a caller concern.
3. A caller checks the error before reading the value: `if (result.error case final error?) { ... }`
   for a named record, `if (result.$2 case final error?) { ... }` for a positional one (or the
   equivalent `switch`/pattern match) is the checkable form. Reading the value without a preceding
   error check is the Dart analogue of ignoring a Go `err` return.
4. Throwing stays for a genuinely unrecoverable programmer error (a broken invariant, a state the code
   proves unreachable) - the Dart analogue of Go's `panic` - and at a boundary a framework requires to
   throw (a `build()` override, a `fromJson` factory a codegen tool calls, an `Isolate` entry point
   whose failure protocol is an uncaught exception). Name the boundary in a comment when it is not
   obvious why a result record was not used there.
5. The error type carried in the record's error field is the `sealed DomainError` hierarchy from
   [[dart-style]] - this rule changes delivery (returned, not thrown), never the type itself.
{{end}}

{{define "forbidden"}}
- `throw`ing a `DomainError` (or any recoverable failure) instead of returning it in the record.
- A result record with both the value and the error non-null, or both null.
- Reading the value from a result record with no preceding check of the error.
- A `try`/`catch` at the call site standing in for checking the record's error.
- A new ad-hoc error type in the error position instead of the `sealed DomainError` hierarchy.
{{end}}

{{define "validation"}}
- [ ] A function that can fail returns `(T?, DomainError?)` or `({T? value, DomainError? error})`
  (or its `Future`), never throws to signal that failure.
- [ ] Exactly one of the value and the error is non-null on any return.
- [ ] Every call site checks the error before reading the value.
- [ ] A remaining `throw` is a broken invariant or a framework-mandated boundary, named as such.
- [ ] The error's type is the `sealed DomainError` hierarchy ([[dart-style]] rule 4), not a
  new ad-hoc error type.
{{end}}
