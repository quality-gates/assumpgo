# Usage

```console
assumpgo <path>...
assumpgo -format xml <path>...
assumpgo -exclude a.go,vendor <path>...
assumpgo -output report.xml -format xml <path>...
assumpgo -version
```

`<path>...` is one or more `.go` files or directories (walked recursively).

`-format` / `-f` accepts exactly `pretty` (the default human table) or `xml`
(checkstyle). Values are case-sensitive and lowercase-only: `-format XML`,
`-format json`, and `-format ""` are usage errors (exit `100`), not a silent
fallback to pretty.

`-output` writes the report to a file instead of stdout. A path that was
collected as an analysis target is refused with exit `100` and left untouched —
`assumpgo -output victim.go victim.go` reports an error rather than replacing
your source with the report.


Example:

```text
$ assumpgo ./mypackage
assumpgo analyser v0.1.1 by quality-gates

-------------------------------------------------
| file        | line | message                  |
=================================================
| dog.go      | 12   | if dog != nil {          |
-------------------------------------------------

1 out of 4 boolean expressions are assumptions (25%)
```

## Exit codes

| Code | Meaning |
| ---: | :--- |
| 0 | No assumptions found |
| 110 | One or more assumptions found |
| 100 | Usage error (e.g. missing path, unknown `-format`) |

## What counts as an assumption

A boolean node is reported as an assumption when it is any of:

| Pattern | Example |
| :--- | :--- |
| A negative comparison `!=` | `dog != nil`, `n != 0` |
| A bare variable used as a condition | `if ready {`, `for running {` |
| Boolean-not of a variable | `!ready` |
| `&&` / `||` mixing a bare variable with a comparison | `x && x == "test"`, `x && y && n == 1` |

Chains of bare variables with no comparison anywhere — `x && y && z`,
`x || y || z` — mix nothing and are **not** flagged. Extra variables on
either side of a real mix do not hide it; operand order and parentheses do
not change the result. Two comparisons (`x == 1 && y == 2`) are not a mix.

The **denominator** (boolean expressions) counts every `if`, every `for` with a
condition, and every `&&` / `||`. Because `!=` and `!var` are flagged wherever
they appear — not only inside those contexts — they count as boolean
expressions too, keeping the percentage a real 0–100 ratio of boolean
expressions that are assumptions.

## How this maps from PHP

php-assumptions flags the loose `==`, the loose `!=`, and the strict-negative
`!==`, but deliberately **not** the strict-positive `===`. Go has a single,
strict set of comparison operators, so:

- Go's `==` is the analog of PHP's `===` (strict positive) — treated as an
  **assertion**, so it is **not** flagged. This includes `x == nil`, the
  idiomatic early-return guard.
- Go's `!=` is the negative, blacklisting comparison the blog post warns about
  — it **is** flagged.

The idiomatic comma-ok assertion — type assertion (`if v, ok := x.(*T); ok`),
map index (`if v, ok := m[k]; ok`), or channel receive (`if v, ok := <-ch; ok`),
including their inverted guards (`!ok`) — binds its variable in the `if` or
`for` init statement and is therefore **not** treated as a bare-variable
assumption. This remains true when `ok` or `!ok` is combined with another
condition, such as `ok && v != nil`; the separate `!=` comparison is still
reported.
