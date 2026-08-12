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
