package assumpgo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCollectGoFilesWithGOFLAGS(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"base.go":    "package p\n",
		"other.go":   "//go:build other\n\npackage p\n",
		"special.go": "//go:build special\n\npackage p\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		flags string
		want  []string
	}{
		{"", []string{"base.go"}},
		{"-tags=special", []string{"base.go", "special.go"}},
		{"--tags=other,special", []string{"base.go", "other.go", "special.go"}},
		{"'-tags=other special'", []string{"base.go", "other.go", "special.go"}},
		{"\t-mod=readonly\n\"-tags=special\"\r", []string{"base.go", "special.go"}},
		{"-tags='special'", []string{"base.go", "special.go"}},
		{"-tags=special -tags=other", []string{"base.go", "other.go"}},
		{"-tags=special -tags=", []string{"base.go"}},
	} {
		t.Run(tc.flags, func(t *testing.T) {
			t.Setenv("GOFLAGS", tc.flags)
			files, err := CollectGoFiles(dir)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, file := range files {
				names = append(names, filepath.Base(file))
			}
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatalf("files = %v, want %v", names, tc.want)
			}
		})
	}
}

func TestCollectGoFilesRejectsMalformedGOFLAGS(t *testing.T) {
	dir := t.TempDir()
	for _, flags := range []string{"-tags", "--tags", "'-tags=special", "-tags='special"} {
		t.Run(flags, func(t *testing.T) {
			t.Setenv("GOFLAGS", flags)
			_, err := CollectGoFiles(dir)
			if err == nil || !strings.Contains(err.Error(), "GOFLAGS") {
				t.Fatalf("error = %v, want a GOFLAGS error", err)
			}
		})
	}
}
