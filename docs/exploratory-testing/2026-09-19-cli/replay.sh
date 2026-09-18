#!/usr/bin/env bash
# Replays confirmed findings from the 2026-09-19 exploratory pass.
# Usage: replay.sh /path/to/assumpgo-binary
set -u
BIN=${1:?assumpgo binary}
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
cd "$W"

echo "== F1: ./... walks hidden, _-prefixed, testdata and vendor dirs"
mkdir -p f1 && cd f1
for d in . .hidden _skip testdata vendor/example.com/dep; do
  mkdir -p "$d"
  printf 'package p\n\nfunc F(x *int) bool {\n\tif x != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n' > "$d/f.go"
done
"$BIN" -format xml ./... | grep -o 'name="[^"]*"'
printf 'package broken\nfunc {\n' > testdata/broken.go
"$BIN" ./... >/dev/null; echo "with broken testdata fixture: exit=$?"
cd "$W"

echo "== F2: file symlink resolves constants from the link's directory"
mkdir -p f2/real f2/other && cd f2
printf 'package p\n\nconst Debug = false\n' > real/b.go
printf 'package p\n\nfunc F() {\n\tif Debug {\n\t}\n}\n' > real/a.go
ln -s ../real/a.go other/a.go
"$BIN" real/a.go >/dev/null; echo "real/a.go exit=$?"
"$BIN" other/a.go >/dev/null; echo "other/a.go (symlink) exit=$?"
cd "$W"

echo "== F3: //line directive rewrites reported line and blanks the message"
mkdir -p f3 && cd f3
printf 'package p\n\nfunc F(x *int) bool {\n//line grammar.y:100\n\tif x != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n' > gen.go
"$BIN" -format xml gen.go | grep '<error'
