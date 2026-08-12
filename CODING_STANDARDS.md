# Coding standards

## Tests

- Strongly prefer integration tests and end-to-end tests over unit tests.
- Strongly prefer exercising real system behaviour over "the tests pass so it must work."
- Only mock third-party services we cannot control. Do not mock code we own.
- For this codebase, the default proof is: run the real analyser/CLI on fixture or real Go source and assert which assumptions are reported, percentages, exit codes, and output shapes.

## Comments and docs

- Code comments use ASD-STE100 Simplified Technical English.
- Ground terms in `CONTEXT.md` domain language when that file exists. Do not invent synonyms for glossary terms.
- Do not write comments that only repeat what the code already makes clear.
- Do not put brittle references in README or comments (versions, line numbers, temporary paths, "as of today" claims) when those details are allowed to change.

## Common footguns

- Tautological tests (asserting the mock was called the way the test just configured it).
- Mocks of modules/services we own.
- "Green suite" treated as proof the product works for a user.
- Narrating comments and README drift magnets.
- Cheating complexity or quality gates with denser syntax, hidden branching, or indirection that does not reduce real complexity.
- "Fixing" `testdata/fixtures/` to look idiomatic — those files are deliberate analyser inputs, ignored by the Go toolchain.

## Go

- Format with `gofmt -s`; keep `go vet` clean. Stay under the static gates CI enforces (`gocyclo -over 15`, `ineffassign`).
- Keep the library flat at module root (`detector`, `analyser`, `output`, `files`) plus `cmd/assumpgo`. Do not introduce an `internal/` split without a clear boundary reason.
- Parse with `go/parser` + `ast.Inspect` through the existing `Analyser` / `Detector` types. Do not add a second analysis stack.
- Detection rules are the public contract. Changing what counts as an assumption requires updating tests and `README.md` in the same change.
- Deliberate php-assumptions divergence: Go `==` is strict — positive equality (including `x == nil`) is an assertion and must **not** be flagged; only negative `!=` (and the other documented bare-variable cases) are. Do not "fix" this.
- Comma-ok / init-bound variables in `if`/`for` are assertions, not assumptions — preserve that behaviour.
- Mutation gate via mutago: overall MSI ≥ 75%, covered-code MSI ≥ 80% on the root package (not `cmd/`). Prefer tests that kill escapes over lowering thresholds; leave documented equivalent mutants alone.
- If a mutago run is interrupted, `git restore` any source left mutated. `report.json` is an artifact and must stay gitignored.
- Smoke-test the binary on fixtures (e.g. `testdata/fixtures/dog.go`) and treat user-visible rows/exit codes as behaviour.
- Keep the `version` constant in `cmd/assumpgo/main.go` aligned when cutting a release.
