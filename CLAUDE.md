# assumpgo

Static analysis for Go that finds weak **assumptions** in boolean checks
(negative `!=` comparisons, bare-variable truthiness) and reports what fraction
of a codebase's boolean expressions are assumptions rather than assertions. A Go
port of [rskuipers/php-assumptions](https://github.com/rskuipers/php-assumptions).

## Build & test

```bash
go build ./cmd/assumpgo
go test ./...
```

All packages pass. The files under `testdata/fixtures/` are valid Go used as
analyser *inputs* — they are deliberately un-idiomatic and are ignored by the Go
toolchain (the `testdata` directory is never built), so do not try to "fix" them.

## Key files

The tool is one small library package (`github.com/quality-gates/assumpgo`) plus
a CLI.

| File | What it does |
| :--- | :--- |
| `detector.go` | `Detector`: `Scan` (is a node an assumption?) and `IsBoolExpression` (the percentage denominator). Holds all the detection rules. |
| `analyser.go` | `Analyser` walks files via `go/parser`+`ast.Inspect`; `Result` collects assumptions and computes `Percentage()`. |
| `output.go` | `Output` interface, `PrettyOutput` (aligned table) and `XMLOutput` (checkstyle XML). |
| `files.go` | `CollectGoFiles` / `CollectFromList` — path discovery for targets and `--exclude`. |
| `cmd/assumpgo/main.go` | Binary entrypoint; all flag wiring, exit codes, and the version constant. |
| `testdata/fixtures/` | Valid Go fixtures (`example.go`, `dog.go`, `cat.go`) used to calibrate the analyser. |

## What counts as an assumption

These rules are the public contract — changing them changes results, so update
the tests and `README.md` together. A node is an assumption when it is:

- a `!=` comparison (e.g. `x != nil`);
- a bare variable used as a condition (`if x`, `for x`), *unless* the variable
  is bound in the statement's init (the comma-ok idiom `if v, ok := x.(*T); ok`
  is an assertion, not an assumption);
- a boolean-not of a variable (`!x`);
- a `&&` / `||` mixing a bare variable with a comparison (`x && x == "test"`).

**Deliberate divergence from php-assumptions:** Go's `==` is strict (the analog
of PHP's `===`, which php-assumptions does *not* flag), so positive equality —
including `x == nil` — is treated as an assertion and is **not** flagged. Only
the negative `!=` is. Do not "fix" this to flag `==`; it is intentional and is
asserted by `TestScanIgnoresStrictEquality`.

## Self-mutation and quality gates

`.github/workflows/mutation.yml` runs [mutago](https://github.com/quality-gates/mutago)
on assumpgo with two hard gates, the same standards mutago holds itself to:

| Gate | Threshold | Flag |
| :--- | :--- | :--- |
| Overall MSI | ≥ 75% | `--min-msi 75` |
| Covered-code MSI | ≥ 80% | `--min-covered-msi 80` |

**Run the gates locally before committing.** Install mutago (CI installs it the
same way), then run against the same package CI uses:

```bash
go install github.com/quality-gates/mutago/v2/cmd/mutago@latest
"$(go env GOPATH)/bin/mutago" \
  --exec-timeout 30 --coverage --min-msi 75 --min-covered-msi 80 \
  github.com/quality-gates/assumpgo
```

mutago's exit code 4 means a gate failed (escaped mutants dropped the score
below a threshold); exit 0 means all gates passed. Only the root package is
mutated — `cmd/` is excluded because its behaviour is integration-, not
unit-, tested.

A run writes a `report.json` artifact into the working directory; it is
gitignored. mutago mutates the package's source files in place and restores
them when finished — if a run is interrupted, check `git status` and
`git restore` any source file left mutated.

## Shipping workflow

Follow these steps in order when landing a change:

1. **Build and test locally** — `go build ./...` and `go test ./...`.
2. **Run the static gates** — these mirror the Go Report Card workflow and must
   be clean: `gofmt -s -l .` (no output), `go vet ./...`, `gocyclo -over 15 .`
   (no output), `ineffassign ./...`.
3. **Run the mutation gate** — the command in the section above. Exit 0 = pass,
   exit 4 = escaped mutants. If a mutant escaped, prefer adding a test that
   kills it over weakening the gate. Some escapes are genuinely unkillable
   *equivalent* mutants; leave those and rely on the thresholds. Known examples:
   error guards on writes to an in-memory buffer (which never error), and
   mutago's `composite/field-clear` clearing the `detector` field in
   `NewAnalyser` — `Detector` is a stateless empty struct whose methods never
   dereference the receiver, so a nil detector is behaviourally identical.
4. **Manual smoke test** — build the binary and actually run it. Do not skip
   this:
   ```bash
   go build -o /tmp/assumpgo ./cmd/assumpgo
   /tmp/assumpgo testdata/fixtures/dog.go   # expect one `if dog != nil {` row, exit 110
   ```
5. **Update docs if needed** — if your change adds, removes, or renames a flag
   or changes user-facing behaviour or the detection rules, update `README.md`
   to match before committing.
6. **Commit and push via a PR** — work on a feature branch and open a PR; the
   gates run on the PR. `main` is not currently push-protected, but land changes
   through a PR anyway so CI verifies them first. Fix forward — no `--force-push`
   and no `--amend` on published commits; if a check fails, fix it in a new
   commit.
7. **Watch CI** — run `gh pr checks <number>` and wait for every workflow
   (Mutation Testing, Security, Go Report Card) to go green. Do not merge if any
   is red.
8. **Squash-merge to main, then sync** — `gh pr merge <number> --squash
   --delete-branch`, then `git checkout main && git pull`.

The release version is the `version` constant in `cmd/assumpgo/main.go`; bump it
when cutting a release.

## Conventions

- **The assumpgo binary's own exit codes** (distinct from mutago's gate exit
  code 4): `0` no assumptions found, `110` one or more assumptions found, `100`
  usage error. The `110` code is what makes the tool usable as a CI gate.
- **Tests are white-box** (`package assumpgo`), so they can exercise unexported
  helpers like `readLine` and `addAssumption` directly. Keep them that way.
- **Edit files one at a time using Read then Edit.** Do not use scripts or
  string-replacement to make the same change across many files at once; small
  per-file differences mean a bulk approach produces inconsistent output.

## Testing posture

- `detector_test.go` parses snippets with the `parseExpr` / `parseStmt` helpers
  and asserts `Scan` / `IsBoolExpression` directly. When you add a detection
  rule, add both a positive case and a negative case (a near-miss that must
  *not* be flagged), because mutation testing rewards asymmetric coverage.
- `analyser_test.go` asserts **exact** line numbers, messages, and counts
  against `testdata/fixtures/example.go`. These hardcoded expectations are the
  point — the fixture is a fixed input. **If you edit a fixture, update the
  expected line numbers and counts in the test**, or the assertions (and the
  documented calibration: 4 assumptions / 7 boolean expressions / 57%) will
  drift.
- `output_test.go` does not hardcode exact column widths; it asserts that all
  table lines share a width (alignment) plus the exact summary line. Follow that
  pattern so cosmetic layout changes don't require golden-file churn.
- The mutation run does **not** touch `testdata/fixtures/`, so there is no
  fixture to restore after `go test` (unlike a tool that mutates an example
  package). Just confirm `git status` is clean.
