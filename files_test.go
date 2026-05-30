package assumpgo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectGoFilesSingleFile(t *testing.T) {
	file := filepath.Join("testdata", "fixtures", "dog.go")
	got, err := CollectGoFiles(file)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	if len(got) != 1 || got[0] != file {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", file, got, file)
	}
}

func TestCollectGoFilesDirectory(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures")
	got, err := CollectGoFiles(dir)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}

	want := map[string]bool{
		filepath.Join(dir, "cat.go"):     false,
		filepath.Join(dir, "dog.go"):     false,
		filepath.Join(dir, "example.go"): false,
	}
	for _, p := range got {
		if !strings.HasSuffix(p, ".go") {
			t.Errorf("collected non-.go file: %q", p)
		}
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("expected %q to be collected", p)
		}
	}
}

func TestCollectGoFilesIgnoresNonGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("nope\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := CollectGoFiles(dir)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	if len(got) != 1 || !strings.HasSuffix(got[0], "keep.go") {
		t.Errorf("CollectGoFiles = %v, want only keep.go", got)
	}
}

func TestCollectGoFilesSingleNonGoFile(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(txt, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An explicitly named file is returned as-is, even without a .go suffix.
	got, err := CollectGoFiles(txt)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	if len(got) != 1 || got[0] != txt {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", txt, got, txt)
	}
}

func TestCollectGoFilesMissingPath(t *testing.T) {
	if _, err := CollectGoFiles(filepath.Join("testdata", "nope")); err == nil {
		t.Error("expected an error for a missing path")
	}
}

func TestCollectFromListEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", " , "} {
		got, err := CollectFromList(in)
		if err != nil {
			t.Fatalf("CollectFromList(%q): %v", in, err)
		}
		if len(got) != 0 {
			t.Errorf("CollectFromList(%q) = %v, want empty", in, got)
		}
	}
}

func TestCollectFromList(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures")
	cat := filepath.Join(dir, "cat.go")
	dog := filepath.Join(dir, "dog.go")

	got, err := CollectFromList(cat + " , " + dog)
	if err != nil {
		t.Fatalf("CollectFromList: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("CollectFromList = %v, want 2 files", got)
	}
	if got[0] != cat || got[1] != dog {
		t.Errorf("CollectFromList = %v, want [%q %q]", got, cat, dog)
	}
}

func TestCollectFromListPropagatesError(t *testing.T) {
	if _, err := CollectFromList("does-not-exist.go"); err == nil {
		t.Error("expected an error for a missing entry in the list")
	}
}
