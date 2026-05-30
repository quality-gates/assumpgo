# assumpgo

Static analysis for Go that finds weak **assumptions** in your boolean checks
and reports how many of your boolean expressions are assumptions rather than
assertions.

It is a Go port of [rskuipers/php-assumptions](https://github.com/rskuipers/php-assumptions),
inspired by the blog post
[*From assumptions to assertions*](https://rskuipers.com/blog/from-assumptions-to-assertions).

## The idea

The blog post argues that negative, "blacklisting" checks are *assumptions*:

```go
if dog != nil {
    dog.Woof() // we ASSUME a non-nil pointer is a usable Dog
}
```

You should *assert* your expectations instead. In Go that means a type
assertion / type switch (the analog of PHP's `instanceof`):

```go
if d, ok := animal.(*Dog); ok {
    d.Woof() // we ASSERT it is a Dog before using it
}
```

`assumpgo` finds the assumptions and tells you what fraction of your boolean
expressions they make up.

## Install

```sh
go install github.com/quality-gates/assumpgo/cmd/assumpgo@latest
```

Or build from a clone:

```sh
go build -o assumpgo ./cmd/assumpgo
```

## Usage

```sh
assumpgo <path>                 # a single .go file or a directory (recursed)
assumpgo -format xml <path>     # checkstyle-style XML, e.g. for CI
assumpgo -exclude a.go,vendor <path>
assumpgo -output report.xml -format xml <path>
assumpgo -version
```

Example:

```
$ assumpgo ./mypackage
assumpgo analyser v0.1.0 by quality-gates

-------------------------------------------------
| file        | line | message                  |
=================================================
| dog.go      | 12   | if dog != nil {          |
-------------------------------------------------

1 out of 4 boolean expressions are assumptions (25%)
```

### Exit codes

| Code | Meaning                          |
|------|----------------------------------|
| 0    | No assumptions found             |
| 110  | One or more assumptions found    |
| 100  | Usage error (e.g. missing path)  |

This makes it usable as a quality gate in CI.

## What counts as an assumption

A boolean node is reported as an assumption when it is any of:

| Pattern                              | Example                     |
|--------------------------------------|-----------------------------|
| A negative comparison `!=`           | `dog != nil`, `n != 0`      |
| A bare variable used as a condition  | `if ready {`, `for running {` |
| Boolean-not of a variable            | `!ready`                    |
| `&&` / `||` mixing a bare variable with a comparison | `x && x == "test"` |

The **denominator** (boolean expressions) counts every `if`, every `for` with a
condition, and every `&&` / `||`.

### How this maps from PHP

php-assumptions flags the loose `==`, the loose `!=` and the strict-negative
`!==`, but deliberately **not** the strict-positive `===`. Go has a single,
strict set of comparison operators, so:

- Go's `==` is the analog of PHP's `===` (strict positive) — treated as an
  **assertion**, so it is **not** flagged. This includes `x == nil`, which is
  the idiomatic early-return guard.
- Go's `!=` is the negative, blacklisting comparison the blog post warns about
  (the `$user !== null` example) — it **is** flagged.

The idiomatic comma-ok assertion (`if v, ok := x.(*T); ok`) binds its variable
in the `if` init statement and is therefore **not** treated as a bare-variable
assumption.

## Development

```sh
go test ./...
go vet ./...
```

The test fixtures in `testdata/fixtures/` are valid Go files used to calibrate
the analyser.

## License

MIT — see [LICENSE](LICENSE).
