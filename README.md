# assumpgo

Catch weak assumptions in Go boolean checks before they calcify: negative
comparisons and bare variables used as conditions where a positive assertion
would pin the intent.

`assumpgo` is a local CLI. It walks Go source, never builds or runs your
project, and needs no project dependencies installed. Go 1.26+. Port of
[php-assumptions](https://github.com/rskuipers/php-assumptions), inspired by
[*From assumptions to assertions*](https://rskuipers.com/blog/from-assumptions-to-assertions).

## Quick start

```console
go install github.com/quality-gates/assumpgo/cmd/assumpgo@latest
assumpgo ./...
```

That scans the tree and prints assumption findings plus the assumption ratio.
Exit `0` is clean, `110` means assumptions found, `100` means usage error.

Common next steps:

```console
assumpgo -format xml ./...
assumpgo -exclude vendor,testdata ./mypackage
assumpgo -output report.xml -format xml ./...
```

Full usage, patterns, and PHP mapping notes: [docs/usage.md](docs/usage.md).

## Install

```console
go install github.com/quality-gates/assumpgo/cmd/assumpgo@latest
assumpgo -version
```

From a local checkout:

```console
go build -o assumpgo ./cmd/assumpgo
```

## Tune the gate

Point at a package or file. Exclude noisy paths with `-exclude`. Prefer
checkstyle XML when a CI system needs a machine report:

```console
assumpgo -format xml -exclude vendor,generated ./...
```

There is no ruleset file: the analyzer’s pattern set is fixed. See
[docs/usage.md](docs/usage.md) for what counts as an assumption.

## Suppress one intentional exception

assumpgo has no per-line disable comment. Skip paths with `-exclude`, or fix
the check to a positive assertion (type assert / comma-ok / positive `==`) so
it is no longer an assumption.

## Drop it into CI

```yaml
# GitHub Actions
- uses: actions/setup-go@v6
  with:
    go-version-file: go.mod
- run: go install github.com/quality-gates/assumpgo/cmd/assumpgo@latest
- run: assumpgo ./...
```

```yaml
# GitLab / generic XML
script: assumpgo -format xml -output gl-assumpgo.xml ./...
```

## Maintainers

Usage reference: [docs/usage.md](docs/usage.md).

Development checks:

```console
go test ./...
go vet ./...
```

CI also runs mutago mutation gates, govulncheck, and Go Report Card A+.

## License

MIT. See [LICENSE](LICENSE).
