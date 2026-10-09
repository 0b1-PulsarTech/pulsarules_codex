---
id: ts-errors
name: Errors - TypeScript result
description: A TypeScript or JavaScript function that can fail returns a result, a tuple with the error last or an object with value and error, and never throws or rejects to signal a recoverable failure; throw stays for a broken invariant or a framework-mandated boundary. The TypeScript and JavaScript delivery counterpart to Go's (T, error); the error type is a closed set of domain errors.
tags:
    - typescript
    - errors
---

# Errors - TypeScript result

> A TypeScript function whose outcome can fail returns a result carrying both the success value and
> the error, either a tuple with the error last or an object `{ value, error }` - never throws or
> rejects to signal that outcome. `throw` stays for a broken invariant (a programmer error with no
> recovery) or a boundary a framework requires to throw.

Applies to: any TypeScript, JavaScript or Svelte (script) function that can fail with a domain
error: validation, a not-found lookup, an operation the caller might retry with different input.
TypeScript and JavaScript have no `throws` clause and no checked exceptions, so a thrown error or a
rejected promise is invisible in the signature: nothing in the type tells a caller the call can
fail, and nothing forces them to handle it. This is the TypeScript delivery counterpart to
[[errors]]'s `(T, error)` return - the principle (failure is part of the return type, not a side
channel) is the same; only the shape differs. Dart states the same idea in [[dart-errors]].

{{define "when"}}
- Writing a TypeScript or JavaScript function whose outcome can fail with a domain error.
- Deciding whether to throw, reject, or return an error.
- Consuming a result at a call site.
- Wrapping an API that throws (`fetch`, `JSON.parse`, a library call).
- Writing a framework boundary that itself requires a throw (a SvelteKit `load` or action).
{{end}}

{{define "must"}}
1. A function that can fail returns a result in one of two forms, and never throws or rejects to
   signal that failure: a tuple with the error last, typed as a discriminated union,
   `readonly [T, null] | readonly [null, E]`, or an object, `{ value: T; error: null } |
   { value: null; error: E }`. An async function returns `Promise<` that result `>`; the promise
   resolves with the result and does not reject. Pick one form per package and keep it.
2. Exactly one side is non-null on return: `error` null means the value is the result; `error`
   non-null means the value carries no meaning the caller may read. The union type makes a function
   returning both non-null or both null a compile error rather than a convention.
3. A caller checks the error before using the value - narrowing on `error` (`if (error !== null)`
   or `if (result.error !== null)`) is the checkable form. Using the value without a preceding
   `error` check is the TypeScript analogue of ignoring a Go `err` return.
4. Catch at the boundary of an API that throws (`fetch`, `JSON.parse`, a library call) and convert
   to the result there, in one small wrapper; the throw never travels past that wrapper.
5. `throw` stays for a genuinely unrecoverable programmer error (a broken invariant, a state the
   code proves unreachable) - the TypeScript analogue of Go's `panic` - and at a boundary a
   framework requires to throw (SvelteKit's `error()` and `redirect()` in `load` and actions throw
   by design). Name the boundary in a comment when it is not obvious why a result was not used.
6. The error type is a closed set of domain errors - a discriminated union on a `kind` field, or a
   small class hierarchy - the TypeScript analogue of Dart's sealed `DomainError`. This rule
   changes delivery (returned, not thrown), never the type itself.
{{end}}

{{define "forbidden"}}
- `throw`ing a domain error, or returning a rejected `Promise`, for a recoverable failure.
- A result with both sides non-null, or both null.
- Using the value from a result with no preceding check of the error.
- A `try`/`catch` at the call site standing in for checking the result's error.
- A new ad-hoc error type, or a bare `string` or `Error`, in the error position instead of the
  closed set of domain errors.
- A throw that escapes an API wrapper instead of being converted to the result there.
{{end}}

{{define "validation"}}
- [ ] A function that can fail returns a tuple with the error last or `{ value, error }` (or a
  `Promise` of it), and never throws or rejects to signal that failure.
- [ ] Exactly one of the value and the error is non-null on any return, enforced by the type.
- [ ] Every call site checks the error before using the value.
- [ ] Every API that throws is wrapped once, at its boundary, into the result.
- [ ] A remaining `throw` is a broken invariant or a framework-mandated boundary, named as such.
- [ ] The error's type is the closed set of domain errors, not an ad-hoc type.
{{end}}
