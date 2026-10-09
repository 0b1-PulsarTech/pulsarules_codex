# Languages

The style analyzers (single-letter names, import order, function length, nesting, ...) read a
syntax tree. A "syntax provider" builds it per language. What each one needs, and what happens when
it cannot work, is below.

| Language | Extensions | Provider | Needs |
| --- | --- | --- | --- |
| Go | `.go` | built in (`go/parser`) | nothing |
| Dart | `.dart` | a Dart helper program | `dart` on PATH (optional) |
| TypeScript, JavaScript | `.ts .tsx .mts .cts .js .jsx .mjs .cjs` | built in parser | nothing |
| Svelte | `.svelte` | built in parser | nothing |

No Node, npm or bun is ever needed or run.

Go: the function length limit applies to named functions and methods. A function literal, such as
`var Handler = func() { ... }`, is not measured as a function of its own.

## When a file cannot be analysed

Every provider reports through its own analyzer id, `<lang>-syntax` where `<lang>` is `go`,
`dart`, `typescript`, `javascript` or `svelte`:

- `<lang>-syntax/unparsed` (WARNING), once per file: the file does not parse, or went over a safety
  limit. The message carries the reason and line. Make the file parse within the limits; until it
  does, no style rule checks it. A broken Go file also gets a `golangci-lint` typecheck finding.
- `<lang>-syntax/unavailable` (INFO), once per language: "N <lang> files not analysed: <reason>".
  The provider itself could not run (for example `dart` is missing). Text-level analyzers
  (typographic markers, file size, top-of-file, ...) still run.

## Dart

- `dart` must be on PATH. With fvm, make one global, for example `fvm global <version>`. Without
  `dart`, one INFO `dart-syntax/unavailable` is printed and only the Dart syntax based checks are
  skipped.
- The helper is a small Dart program compiled ahead of time (`dart compile exe`) after
  `dart pub get --offline`, falling back to an online `pub get`. The first build therefore needs the
  pub cache to be populated or network access. A cold build takes about 8 seconds; later runs about
  0.2 seconds.
- Cache location: `<user cache dir>/pulsarules/dart-syntax/<sdk>-<os>-<arch>-<hash>`; with
  `XDG_CACHE_HOME` set that is under it. Entries unused for 7 days are pruned.
- Limits: 45 seconds to build, 30 seconds per run (retried once if the executable was pruned
  meanwhile). The Stop hook stops waiting after 60 seconds, so a cold build (up to 45) plus a run
  (up to 30) can exceed it: the notice then says the run did not finish, the run is abandoned and
  may leave the helper running and a request file in the temp directory behind, and the git hooks
  still check.
- A failed build writes `failed-<key>.json` beside the cache and is not retried for 10 minutes
  ("dart helper build failed Ns ago, not retrying for 10m0s: ..."). Fix the cause, then delete the
  `failed-*.json` files or wait 10 minutes.
- Each build leaves a dangling entry in `~/.pub-cache/active_roots`; `dart pub cache gc` cleans it.
- A pathological file (for example tens of thousands of chained `..` cascades) can exceed the 30
  second batch bound and make the whole run `unavailable`.
- Rules: only positional parameters are counted for the parameter limit; `_` is a valid unused
  name. Bound closures and local functions are measured as functions. Switch case bodies count
  for nesting, and `try`/`catch`/`finally` add depth. Test containers (`main`, `group`) are exempt
  from the function length limit.
- Import order is `dart:`, then `package:`, then relative imports, matching the
  `directives_ordering` lint.

## TypeScript, JavaScript and Svelte

- The parser is Go code from the typescript-go fork (`github.com/Zzzen/typescript-go`, Apache-2.0),
  linked into the binary. It runs in a child process: the same `pulsarules_cli` binary started
  again as `__tsproc_child` with `PULSARULES_TSPROC=1`. A child never starts another child.
- Guards per file: 256 MiB memory, 10 seconds, 1 MiB source, plus bounds on the total bracket
  count (32768), the operator count (65536) and the nesting depth (500). A file over a guard is
  skipped and reported as `<lang>-syntax/unparsed` ("skipped: ...").
- If the child cannot start, `typescript-syntax/unavailable`, `javascript-syntax/unavailable` and
  `svelte-syntax/unavailable` (INFO) are reported.
- Svelte: each top-level function in a `<script>` block is measured on its own. Anonymous nested
  closures are not measured separately; named or bound ones are. In test files the `describe`,
  `it`, `test` and `before*`/`after*` callbacks are test containers. The `$:` reactive label is
  exempt from `single-letter-name`.
- Import groups and path rules are described under `import-groups` in `docs/analyzers.md`.

## Go

Go needs no helper. See `docs/analyzers.md` for the Go specific analyzers.
