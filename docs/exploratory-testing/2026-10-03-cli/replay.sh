#!/usr/bin/env bash
# Replays verified scenarios from the 2026-10-03 exploratory pass.
# Usage: replay.sh /path/to/assumpgo-binary
set -u
BIN=${1:?assumpgo binary}
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
cd "$W"

echo "== J1: Comma-ok inheritance and boundaries in else-if chains and calls"
mkdir -p j1 && cd j1
printf 'module example.com/j1\n\ngo 1.26\n' > go.mod
cat > chain.go << 'GO'
package j1

func F(x, y any) {
	if v1, ok := x.(*int); ok {
		_ = v1
	} else if v2, ok := y.(*string); ok {
		_ = v2
	} else if ok {
	}
}
GO
echo "-- else-if comma-ok chain --"
"$BIN" chain.go
echo "exit=$?"

cat > boundaries.go << 'GO'
package j1

func pred(b bool) bool { return b }

func G(x any) {
	if v, ok := x.(*int); ok {
		_ = v
	} else if func() bool { return !ok }() {
	}
}
GO
echo "-- call/closure boundary in else-if --"
"$BIN" boundaries.go
echo "exit=$?"
cd "$W"

echo "== J2: Directory walks, build tags, and non-regular files"
mkdir -p j2/pkg && cd j2
printf 'module example.com/j2\n\ngo 1.26\n' > go.mod
printf 'package j2\nfunc F(x *int) {\n\tif x == nil {\n\t\treturn\n\t}\n}\n' > pkg/main.go
printf '//go:build ignore\n\npackage j2\nfunc broken( {\n}\n' > pkg/skip.go
printf 'package j2\nfunc Win(x *int) {\n\tif x != nil {\n\t\treturn\n\t}\n}\n' > pkg/sys_windows.go
mkfifo pkg/pipe.go 2>/dev/null || true
echo "-- directory walk with ignored files, platform files, and FIFO --"
"$BIN" ./pkg/...
echo "exit=$?"

echo "-- overwrite protection for -output --"
"$BIN" -output pkg/main.go pkg/main.go 2>&1
echo "exit=$?"
cd "$W"

echo "== J3: Pretty table alignment, control escapes, and checkstyle XML"
mkdir -p j3 && cd j3
cat > visual.go << 'GO'
package j3

func V(x any) {
	if x != nil { // 🇬🇧 ❤️ ☕ ䷀ \x1b[31mred\x1b[0m
	}
}
GO
echo "-- pretty output with wide characters and escapes --"
"$BIN" visual.go > visual.out
cat visual.out
python3 - << 'PY'
import ctypes, ctypes.util
from pathlib import Path
libc = ctypes.CDLL(ctypes.util.find_library("c"))
libc.wcswidth.argtypes = [ctypes.c_wchar_p, ctypes.c_size_t]
libc.wcswidth.restype = ctypes.c_int
widths = []
for ln in Path("visual.out").read_text(encoding="utf-8").splitlines():
    if ln.startswith("|") or (ln and set(ln) <= set("-=")):
        w = libc.wcswidth(ln, len(ln))
        widths.append(w)
        print(f"wcswidth={w} {ln}")
assert len(set(widths)) == 1, f"Table misalignment detected: {widths}"
print(f"Alignment verified: all {len(widths)} lines have width {widths[0]}")
PY
echo "-- checkstyle XML validity with xmllint --"
"$BIN" -format xml visual.go | xmllint --noout -
echo "xmllint exit=$?"
cd "$W"
