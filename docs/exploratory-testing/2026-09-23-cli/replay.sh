#!/usr/bin/env bash
# Replays confirmed findings from the 2026-09-23 exploratory pass.
# Usage: replay.sh /path/to/assumpgo-binary
set -u
BIN=${1:?assumpgo binary}
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
cd "$W"

echo "== F1: ./... collects hidden (.-prefixed) and _-prefixed files"
mkdir -p f1 && cd f1
printf 'module example.com/f1\n\ngo 1.26\n' > go.mod
printf 'package p\n\nfunc F(x *int) bool {\n\treturn x == nil\n}\n' > main.go
printf 'package p\n\nfunc Disabled(x *int) bool {\n\tif x != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n' > _disabled.go
printf '\x00\x05\x16\x07\x00\x02\x00\x00Mac OS X        ' > ._main.go

echo "-- go vet ./... (should pass, ignoring ._ and _-prefixed files) --"
go vet ./...; echo "go vet exit=$?"

echo "-- assumpgo ./... with AppleDouble ._main.go present --"
"$BIN" ./... >/dev/null; echo "with ._main.go: exit=$?"

echo "-- assumpgo ./... with only _disabled.go (._main.go removed) --"
rm ._main.go
"$BIN" -format xml ./... | grep '<error'
"$BIN" ./... >/dev/null; echo "with _disabled.go: exit=$?"
cd "$W"

echo "== F1 (cont): scanDirConsts resolves constants from _-prefixed files"
mkdir -p f2 && cd f2
printf 'module example.com/f2\n\ngo 1.26\n' > go.mod
printf 'package p\n\nfunc F() {\n\tif Debug {\n\t}\n}\n' > main.go
printf 'package p\n\nconst Debug = false\n' > _consts.go

echo "-- go build ./... (fails because Debug is in ignored file) --"
go build ./... 2>&1 || echo "go build exit=$?"

echo "-- assumpgo ./... (erroneously treats Debug as package const) --"
"$BIN" ./... >/dev/null; echo "assumpgo exit=$?"
