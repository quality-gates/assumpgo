#!/usr/bin/env bash
set -eu
root=$(git rev-parse --show-toplevel)
work=$(mktemp -d "$root/.explore-20261031.XXXXXX")
cleanup() { chmod -R u+w "$work"; rm -rf "$work"; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
export GOCACHE="$work/cache" GOPATH="$work/gopath" GOMODCACHE="$work/modcache" GOFLAGS=-mod=readonly
mkdir -p "$work/bin" "$work/project/hidden" "$work/project/testdata"
cd "$root"
go build -o "$work/bin/assumpgo" ./cmd/assumpgo
bin="$work/bin/assumpgo"
run() {
  printf '\n$ assumpgo'; printf ' %s' "$@"; printf '\n'
  set +e
  "$bin" "$@"
  code=$?
  set -e
  printf 'exit=%s\n' "$code"
}
printf '%s\n' '=== readiness ==='
run testdata/fixtures/dog.go
cd "$work/project"
cat > go.mod <<'END'
module example.test/explore

go 1.26.6
END
cat > main.go <<'END'
package explore
const Enabled = true
func check(x *int, ready bool, m map[string]int) {
 if Enabled {}
 if x == nil {}
 if ready {}
 if x != nil {}
 if v, ok := m["key"]; ok && v == 1 {} else if !ok {}
}
END
cat > hidden/other.go <<'END'
package hidden
func check(ready bool) { if ready {} }
END
cat > testdata/skipped.go <<'END'
package skipped
func check(ready bool) { if ready {} }
END
printf '%s\n' '=== detection, ordinary and corrected ==='
run main.go
cat > clean.go <<'END'
package clean
func check(x *int) { if x == nil {} }
END
run clean.go
printf '%s\n' '=== discovery, exclusion and alias ==='
run ./...
run -exclude hidden ./...
ln main.go alias.go
run main.go alias.go
printf '%s\n' '=== XML, file output, overwrite protection ==='
run -format xml -output report.xml main.go
python3 -c 'import xml.etree.ElementTree as E; r=E.parse("report.xml").getroot(); print("xml root=",r.tag,"files=",len(r.findall("file")),"errors=",len(r.findall(".//error")))'
cp main.go original.go.txt
run -output alias.go main.go
cmp main.go original.go.txt && echo 'source unchanged'
run -format XML main.go
printf '%s\n' '=== build-tag variation ==='
mkdir tagged
cat > tagged/main.go <<'END'
package tagged
func check(ready bool) { if ready == true {} }
END
cat > tagged/special.go <<'END'
//go:build special

package tagged
func special(ready bool) { if ready {} }
END
printf 'go list default: '; go list -f '{{.GoFiles}}' ./tagged
printf 'go list with GOFLAGS=-tags=special: '; GOFLAGS='-mod=readonly -tags=special' go list -f '{{.GoFiles}}' ./tagged
run ./tagged
printf '\n$ GOFLAGS=-tags=special assumpgo ./tagged\n'
set +e
GOFLAGS='-mod=readonly -tags=special' "$bin" ./tagged
code=$?
set -e
printf 'exit=%s\n' "$code"
run tagged/special.go
