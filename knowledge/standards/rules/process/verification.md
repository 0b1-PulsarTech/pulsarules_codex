---
id: verification
name: Verification before done
description: Nothing is done on a claim - build, race tests, formatting, linters, and governance must be green on the final tree and seen this session; every behavior change proven by a failing-then-passing test; failures and skips reported faithfully, never rounded up to done.
tags:
    - process
    - verification
---

# Verification before done

> A change is done only when the build compiles, the tests pass under the race detector, and the
> linters are clean - and you have seen that output this session, not assumed it. This is the
> definition-of-done gate before reporting complete or writing a commit.

Applies to: finishing any change, before reporting it done or committing.

{{define "when"}}
- About to report a task complete or write a commit.
- Claiming something is fixed, passing, or working.
- Finishing a bug fix or a behavior change.
{{end}}

{{define "must"}}
1. Report only verified state. Never say "done", "fixed", "passing", or "should work" without having
   run the check and read its output this session; if you did not run it, say so.
2. Run the project's full definition-of-done gate before declaring done: a compile/build check, the
   full test suite, and static analysis/lint, all clean. Where a governance pipeline exists, run it
   and treat any error-severity finding as a blocker.
3. Prove behavior changes with a test that fails against the old code; a green suite that never
   touched the change proves nothing.
4. Report faithfully: state failures with their output, state which steps were skipped and why, and
   never round a partial result up to success.
5. Re-verify after the final edit - the gate must pass on the exact tree you are about to commit.
{{end}}

{{define "forbidden"}}
- Claiming "done" / "fixed" / "passing" without having run the check this session.
- Committing or reporting complete while the build, tests, or linters are red.
- Declaring a bug fixed with no test that reproduces it.
- Verifying an earlier state and then editing again without re-running the gate.
{{end}}

{{define "validation"}}
- [ ] Build/compile check, the full test suite, and lint/static analysis were run and read this
  session.
- [ ] Governance (where present) ran clean on the final tree.
- [ ] Every behavior change is covered by a test that fails against the old code.
- [ ] The report matches reality: failures and skips stated, nothing rounded up.
{{end}}
