#!/usr/bin/env bash
set -eu
unset GOFLAGS
root=$(git rev-parse --show-toplevel)
work="$root/.scratch/replay"
bin="$root/.scratch/bin/assumpgo"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT HUP INT TERM
cleanup
mkdir -p "$work/project/sub" "$work/project/testdata" "$work/tagged"
cd "$work"
printf 'package p\nfunc f(ready bool, n int) {\n if ready {}\n if n != 0 {}\n if n == 0 {}\n}\n' > project/main.go
printf 'package p\nfunc clean(n int) { if n == 0 {} }\n' > project/sub/clean.go
printf 'package p\nconst ready = true\nfunc f(ready bool) { if ready {} }\n' > shadow.go
printf 'package ignored\nfunc f(x bool) { if x {} }\n' > project/testdata/ignored.go
printf 'package tagged\nfunc clean(n int) { if n == 0 {} }\n' > tagged/base.go
printf '//go:build special\n\npackage tagged\nfunc tagged(ready bool) { if ready {} }\n' > tagged/special.go
run() {
  expected=$1; shift
  set +e
  "$@" > stdout.txt 2> stderr.txt
  status=$?
  set -e
  printf '$'; printf ' %q' "$@"; printf '\nexit=%s\n' "$status"
  cat stdout.txt stderr.txt
  if [ "$status" -ne "$expected" ]; then echo "expected exit $expected" >&2; exit 1; fi
}
printf '== Gate: normal and corrected input ==\n'
run 110 "$bin" project/main.go
grep -q '2 out of 4 boolean expressions are assumptions' stdout.txt
run 0 "$bin" project/sub/clean.go
grep -q '0 out of 1 boolean expressions' stdout.txt
run 110 "$bin" shadow.go
grep -q '1 out of 1 boolean expressions' stdout.txt
run 110 "$bin" "$root/testdata/fixtures/dog.go"
grep -q 'if dog != nil' stdout.txt
printf '== Selection: pattern, exclusion, and build tags ==\n'
run 110 "$bin" 'project/...'
if grep -q ignored.go stdout.txt; then echo 'testdata was not skipped' >&2; exit 1; fi
grep -q '2 out of 5 boolean expressions' stdout.txt
run 0 "$bin" -exclude project/main.go 'project/...'
run 0 "$bin" tagged
run 110 env GOFLAGS=-tags=special "$bin" tagged
grep -q special.go stdout.txt
printf '== CI output: XML, invalid format, overwrite protection ==\n'
run 110 "$bin" -format xml -output report.xml project/main.go
[ ! -s stdout.txt ]
python3 -c 'import xml.etree.ElementTree as E; r=E.parse("report.xml").getroot(); assert r.tag=="checkstyle"; e=r.findall("./file/error"); assert len(e)==2; print("XML parsed: 1 file, 2 findings, lines", ",".join(x.attrib["line"] for x in e))'
run 100 "$bin" -format XML project/main.go
cp project/main.go original.go
ln project/main.go alias.go
run 100 "$bin" -output alias.go project/main.go
cmp original.go project/main.go
printf 'source unchanged after hard-link overwrite attempt\n'
printf '== Completed ==\n'
