# Usage

```console
assumpgo <path>...
assumpgo -format xml <path>...
assumpgo -exclude a.go,vendor <path>...
assumpgo -output report.xml -format xml <path>...
assumpgo -version
```

`<path>...` is one or more `.go` files or directories (walked recursively).


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
| 100 | Usage error (e.g. missing path) |

## What counts as an assumption

A boolean node is reported as an assumption when it is any of:

| Pattern | Example |
| :--- | :--- |
| A negative comparison `!=` | `dog != nil`, `n != 0` |
| A bare variable used as a condition | `if ready {`, `for running {` |
| Boolean-not of a variable | `!ready` |
| `&&` / `||` mixing a bare variable with a comparison | `x && x == "test"` |

Chains of bare variables with no comparison anywhere — `x && y && z`,
`x || y || z` — mix nothing and are **not** flagged.

The **denominator** (boolean expressions) counts every `if`, every `for` with a
condition, and every `&&` / `||`.

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
assumption.

