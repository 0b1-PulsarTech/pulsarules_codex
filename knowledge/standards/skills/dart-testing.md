---
id: dart-testing
name: Dart testing
---

Governs how Dart tests are written: one `_test.dart` per source file, colocated by mirroring the
`lib/src` path under `test/`; table-driven cases named `testCases`/`testCase` (never `tt`);
`package:test`, never `package:flutter_test`; and coverage of the success path plus every failure
variant a unit can produce, each asserted on the returned error rather than a single generic
any-error assertion. Reach for it whenever a `_test.dart` is touched, including a first test for a
source file that has none yet. It is the direct Dart analogue of this repo's `integration-tests`,
carrying the same `testCases`/`testCase` table shape and every-failure-path discipline across
languages.

The rules below are the composed dart-testing rule.
