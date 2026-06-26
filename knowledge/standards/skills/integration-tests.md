---
id: integration-tests
name: Integration tests
---

Governs how tests are written: colocated same-package tests, one `_test.go` per source file,
table-driven cases covering success and every failure path, real-DB repository tests behind a build
tag, and E2E tests that boot the real server. Despite the name, this is not only for cross-boundary
tests - reach for it whenever any `_test.go` is touched, including plain unit tests for business
logic. Composes colocated-mocks for test doubles and fixture-builder for shared fixtures; pairs with
concurrency's deterministic-time guidance so tests never reach for `time.Sleep`.

The rules below are the composed integration-tests rule.
