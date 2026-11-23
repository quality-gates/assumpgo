package assumpgo

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

type countingConstLookup struct{ calls int }

func (c *countingConstLookup) IsPackageConst(_, _, _ string) bool {
	c.calls++
	return false
}

func TestLogicalChainResolutionScalesLinearly(t *testing.T) {
	for _, init := range []string{"", "_, ok := m[key]; "} {
		for _, size := range []int{32, 128} {
			chain := strings.Repeat("flag && ", size-1) + "flag"
			if init != "" {
				chain = "ok && " + chain
			}
			src := "package p\nfunc f() { if " + init + chain + " {} }\n"
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			lookup := &countingConstLookup{}
			result := &Result{}
			err = NewAnalyser(NewDetector(), nil).analyseFile(parsedSource{path: "p.go", src: []byte(src), fset: fset, file: file}, result, lookup)
			if err != nil {
				t.Fatal(err)
			}
			if lookup.calls > 8*size {
				t.Errorf("init %q size %d: %d identifier lookups exceed linear bound %d", init, size, lookup.calls, 8*size)
			}
		}
	}
}
