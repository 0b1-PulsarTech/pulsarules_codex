---
id: dart-testing
name: Dart testing
description: One _test.dart per source file, colocated by mirroring the lib/src path; table-driven cases named testCases/testCase (never tt); package:test, never package:flutter_test; cover the success path and every DomainError variant a unit can produce.
tags:
    - dart
    - testing
references:
    - dart-test-package
---

# Dart testing

> One `_test.dart` per source file, colocated by mirroring the `lib/src` path under `test/`;
> table-driven cases named `testCases`/`testCase` (never `tt`); `package:test`, never
> `package:flutter_test`; cover the success path and every failure path a unit can produce.

Applies to: every `_test.dart` file in a project where Dart is the authorial language. The Dart
analogue of [[testing]] and [[integration-tests]].

{{define "when"}}
- Writing a unit test for a Dart function, class, or use case.
- Adding a new `.dart` source file that has no test yet.
- Testing a function or use case that can produce more than one failure variant.
- Deciding what a table-driven test's loop variable and collection are named.
{{end}}

{{define "must"}}
1. One `_test.dart` per source file: `lib/src/<path>/<name>.dart` is tested by
   `test/<path>/<name>_test.dart` - the test path mirrors the source path with the `src/` segment
   dropped and `_test` appended to the file name. A source file with no corresponding test file is an
   untested unit, not an exempt one.
2. Depend on `package:test`, never `package:flutter_test` - a project whose Dart target is `dart2js`/
   server-side, with no Flutter widget tree, has nothing for `flutter_test` to pump or find.
3. Default to a table-driven shape. The collection is named `testCases`, the loop variable is named
   `testCase` - NEVER `tt` or `tests`. A `testCases` list is a `List` of a positional or named record
   (`(name: ..., input: ..., want: ...)`) or a small local class; each row's `name` (or its equivalent
   first field) becomes the `test()` description, so a failing case is identifiable by name alone.
4. Cover the success path and every failure path a unit can produce. Where the unit under test can
   fail with more than one variant of a sealed error hierarchy (see [[dart-style]]), a `testCases`
   table drives at least one case per variant, asserted on the error the function returns
   (`expect(result.error, isA<Variant>())`, or an exact equality match against the sealed value),
   per [[dart-errors]]; never a single "it returns some error" assertion that would pass for the
   wrong variant. `throwsA` is for a broken-invariant throw only.
5. Name the outer `group()` after the symbol under test, or `Type.method` for a single method's
   suite; keep each `test()` description a readable phrase describing the scenario, never a coded id.
6. Run `dart test` (via the project's `task check`/`task toolchain:check` entrypoint) before reporting
   any Dart change done; a change that touches a `sealed` hierarchy or its exhaustive `switch`
   re-verifies every switch site the compiler now flags.
{{end}}

{{define "examples"}}
Table-driven unit test (collection `testCases`, loop variable `testCase`, `test()` description taken
from the row's `name`):

```dart
import 'package:test/test.dart';

void main() {
  group('ParseStatus', () {
    final testCases = [
      (name: 'draft', input: 'draft', want: Status.draft),
      (name: 'closed', input: 'closed', want: Status.closed),
    ];

    for (final testCase in testCases) {
      test(testCase.name, () {
        expect(parseStatus(testCase.input), testCase.want);
      });
    }
  });
}
```

Covering every failure variant of a sealed error hierarchy:

```dart
final testCases = [
  (name: 'no session', stub: _Stub(error: const AuthError('x')), want: isA<AuthError>()),
  (name: 'not permitted', stub: _Stub(error: const ForbiddenError('x')), want: isA<ForbiddenError>()),
  (name: 'bad input', stub: _Stub(error: const ValidationError('x')), want: isA<ValidationError>()),
];

for (final testCase in testCases) {
  test(testCase.name, () {
    expect(loadThing(testCase.stub).error, testCase.want);
  });
}
```
{{end}}

{{define "forbidden"}}
- A `.dart` source file under `lib/` with no matching `_test.dart`.
- `package:flutter_test` in a project with no Flutter target.
- Naming the table collection/row anything other than `testCases`/`testCase` (`tt`, `tests`, `cases`).
- A single "returns some error" assertion standing in for coverage of a unit's several failure
  variants.
- A `test()` description that is a coded id instead of a readable scenario phrase.
{{end}}

{{define "validation"}}
- [ ] Every source file has a colocated `_test.dart` mirroring its `lib/src` path.
- [ ] Tests depend on `package:test`, not `package:flutter_test`.
- [ ] Table-driven tests use `testCases`/`testCase`, never `tt`/`tests`.
- [ ] The success path and every failure/error variant the unit can produce are each covered by a
  case asserted on the returned error, not a single generic any-error assertion.
- [ ] `group()` is named after the symbol/method under test; `test()` descriptions read as scenarios.
- [ ] `dart test` passes before the change is reported done.
{{end}}
