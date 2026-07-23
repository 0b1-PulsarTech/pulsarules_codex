---
id: verification-before-done
name: Verification before done
---

Nothing is "done" on a claim. A change is done only when the build compiles, the tests pass under the
race detector, and the linters are clean - and you have seen that output this session, not assumed it.
The router loads this last, alongside `commits`, as the definition-of-done gate before you report a
task complete or write a commit. In this repo the governance pipeline
(`pulsarules_cli governance --project .`) is a machine gate for the same contract.

The gate - run build/vet/`-race`/`gofmt`/lint and read the output, prove behavior with a
failing-then-passing test, report failures and skips faithfully, re-verify the final tree - is the
composed verification rule.

## Mandatory workflow

1. Report only VERIFIED state. Never say "done", "fixed", "passing", or "should work" without having
   run the check and read its output this session; if you did not run it, say so.
2. Run the full gate before declaring done: `go build ./...`, `go vet ./...`, `go test -race ./...`,
   `gofmt -l` clean, and the project linter (`task lint` / `golangci-lint run`). Where a governance
   pipeline exists, run it and treat any error-severity finding as a blocker.
3. Prove behavior changes with a test that fails against the OLD code; a green suite that never
   touched the change proves nothing.
4. Report faithfully: state failures with their output, state which steps were skipped and why, and
   never round a partial result up to success.
5. Re-verify after the FINAL edit - the gate must pass on the exact tree you are about to commit.

## Validation checklist

- [ ] Build, vet, `-race` tests, `gofmt -l`, and the linter were run and read this session.
- [ ] Governance (where present) ran clean on the final tree.
- [ ] Every behavior change is covered by a test that fails against the old code.
- [ ] The report matches reality: failures and skips stated, nothing rounded up.

## Forbidden actions

- Claiming "done" / "fixed" / "passing" without having run the check this session.
- Committing or reporting complete while the build, tests, or linters are red.
- Declaring a bug fixed with no test that reproduces it.
- Verifying an earlier state and then editing again without re-running the gate.

## Expected outputs

- A green build, `-race` test run, formatter, and linter on the exact tree being committed.
- A report that names every failure and every skipped step.
