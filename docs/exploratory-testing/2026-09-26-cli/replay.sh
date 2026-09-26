#!/usr/bin/env bash
# Replays confirmed findings from the 2026-09-26 exploratory pass.
# Usage: replay.sh /path/to/assumpgo-binary
set -u
BIN=${1:?assumpgo binary}
W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
cd "$W"

echo "== F1: ./... analyses files the go tool ignores"
mkdir -p f1 && cd f1
printf 'module example.com/f1\n\ngo 1.26\n' > go.mod
printf 'package f1\n\nfunc F(x *int) {\n\tif x == nil {\n\t\treturn\n\t}\n}\n' > main.go
printf 'package f1\n\nfunc broken( {\n}\n' > sys_windows.go
echo "-- go vet ./... (darwin; sys_windows.go is not in the build) --"
go vet ./... >/dev/null 2>&1
echo "go vet exit=$?"
echo "-- assumpgo ./... --"
"$BIN" ./... >/dev/null
echo "assumpgo exit=$?"
printf '//go:build ignore\n\npackage f1\n\nfunc broken( {\n}\n' > skip.go
rm -f sys_windows.go
echo "-- go vet ./... with //go:build ignore syntax error --"
go vet ./... >/dev/null 2>&1
echo "go vet exit=$?"
echo "-- assumpgo ./... --"
"$BIN" ./... >/dev/null
echo "assumpgo exit=$?"
rm -f skip.go
printf 'package f1\n\nfunc Win(x *int) {\n\tif x != nil {\n\t\treturn\n\t}\n}\n' > sys_windows.go
echo "-- valid windows-only assumption; go vet then assumpgo --"
go vet ./... >/dev/null 2>&1
echo "go vet exit=$?"
"$BIN" ./... >/dev/null
echo "assumpgo exit=$?"
"$BIN" ./... | sed -n '1,12p'
cd "$W"

echo "== F2: const in a build-excluded file hides a variable assumption"
mkdir -p f2 && cd f2
printf 'module example.com/f2\n\ngo 1.26\n' > go.mod
printf 'package f2\n\nfunc F() {\n\tif Enabled {\n\t\tprintln(1)\n\t}\n}\n' > use.go
printf '//go:build windows\n\npackage f2\n\nconst Enabled = true\n' > enabled_windows.go
printf '//go:build !windows\n\npackage f2\n\nvar Enabled = true\n' > enabled_other.go
echo "-- go vet ./... --"
go vet ./... >/dev/null 2>&1
echo "go vet exit=$?"
echo "-- assumpgo use.go (Enabled is a var on darwin) --"
"$BIN" use.go >/dev/null
echo "assumpgo exit=$?"
echo "-- same file after removing the windows-only const --"
rm enabled_windows.go
"$BIN" use.go >/dev/null
echo "assumpgo exit=$?"
cd "$W"

echo "== F3: pretty table rows are narrower than the border for emoji"
mkdir -p f3 && cd f3
python3 - << 'PY'
from pathlib import Path
Path("flag.go").write_text("package f3\n\nfunc F(x *int) {\n\tif x != nil { // 🇬🇧\n\t}\n}\n", encoding="utf-8")
Path("heart.go").write_text("package f3\n\nfunc F(x *int) {\n\tif x != nil { // ❤️\n\t}\n}\n", encoding="utf-8")
PY
"$BIN" flag.go > flag.out
"$BIN" heart.go > heart.out
python3 - << 'PY'
import ctypes, ctypes.util
from pathlib import Path
libc = ctypes.CDLL(ctypes.util.find_library("c"))
libc.wcswidth.argtypes = [ctypes.c_wchar_p, ctypes.c_size_t]
libc.wcswidth.restype = ctypes.c_int
for name in ("flag.out", "heart.out"):
    print(f"-- {name} --")
    for ln in Path(name).read_text(encoding="utf-8").splitlines():
        if ln.startswith("|") or (ln and set(ln) <= set("-=")):
            print(f"wcswidth={libc.wcswidth(ln, len(ln))} {ln}")
PY
cd "$W"

echo "== F4: pretty output contains raw ESC from the source line"
mkdir -p f4 && cd f4
printf 'package f4\n\nfunc F(x *int) {\n\tif x != nil { // \033[31mX\033[0m\n\t}\n}\n' > ansi.go
"$BIN" ansi.go > ansi.out
python3 - << 'PY'
from pathlib import Path
data = Path("ansi.out").read_bytes()
print("esc_bytes", data.count(b"\x1b"))
for ln in data.splitlines():
    if b"ansi.go" in ln:
        print(ln)
PY
