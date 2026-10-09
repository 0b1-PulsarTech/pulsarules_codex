---
id: ts-errors
name: TypeScript errors
---

Governs how a TypeScript function delivers failure: a result returned by the function, either a
tuple with the error last or an object `{ value, error }`, never a thrown error or a rejected
promise, because TypeScript carries no `throws` clause and a thrown error is invisible at both the
call site and the compiler. Reach for it whenever writing a TypeScript, JavaScript or Svelte
function that can fail with a domain error, deciding throw-vs-return, wrapping an API that throws,
or consuming a result at a call site. It is the direct TypeScript analogue of this repo's
`errors-logging` and `dart-errors`: the same "failure is part of the return type" principle Go
states as `(T, error)`. It does not redefine the error type beyond requiring a closed set of domain
errors; `throw` stays for a broken invariant or a framework-mandated boundary such as SvelteKit's
`error()` and `redirect()`.

The rules below are the composed ts-errors rule.
