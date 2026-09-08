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

func TestCollectGoFilesCleansPaths(t *testing.T) {
	file := filepath.Join(".", "testdata", "fixtures", "dog.go")
	got, err := CollectGoFiles(file)
	if err != nil {
		t.Fatalf("CollectGoFiles(%q): %v", file, err)
	}
	want := filepath.Join("testdata", "fixtures", "dog.go")
	if len(got) != 1 || got[0] != want {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", file, got, want)
	}

	dir := filepath.Join(".", "testdata", "fixtures")
	gotDir, err := CollectGoFiles(dir)
	if err != nil {
		t.Fatalf("CollectGoFiles(%q): %v", dir, err)
	}
	for _, p := range gotDir {
		if strings.HasPrefix(p, "."+string(filepath.Separator)) {
			t.Errorf("CollectGoFiles(%q) returned uncleaned path %q", dir, p)
		}
	}
}

func TestCollectFromListCleansAndDeduplicates(t *testing.T) {
	in := filepath.Join(".", "testdata", "fixtures", "dog.go") + " , " +
		filepath.Join("testdata", "", "fixtures", "dog.go") + " , " +
		filepath.Join("testdata", "fixtures", "cat.go")
	got, err := CollectFromList(in)
	if err != nil {
		t.Fatalf("CollectFromList(%q): %v", in, err)
	}
	wantDog := filepath.Join("testdata", "fixtures", "dog.go")
	wantCat := filepath.Join("testdata", "fixtures", "cat.go")
	if len(got) != 2 || got[0] != wantDog || got[1] != wantCat {
		t.Errorf("CollectFromList(%q) = %v, want [%q %q]", in, got, wantDog, wantCat)
	}
}

func TestCollectTargetsEmpty(t *testing.T) {
	for _, in := range [][]string{nil, {}} {
		got, err := CollectTargets(in)
		if err != nil {
			t.Fatalf("CollectTargets(%v): %v", in, err)
		}
		if len(got) != 0 {
			t.Errorf("CollectTargets(%v) = %v, want empty", in, got)
		}
	}
}

func TestCollectTargetsMultipleFiles(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures")
	cat := filepath.Join(dir, "cat.go")
	dog := filepath.Join(dir, "dog.go")

	got, err := CollectTargets([]string{cat, dog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("CollectTargets = %v, want 2 files", got)
	}
	if got[0] != cat || got[1] != dog {
		t.Errorf("CollectTargets = %v, want [%q %q]", got, cat, dog)
	}
}

func TestCollectTargetsMultipleDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	f1 := filepath.Join(dir1, "a.go")
	f2 := filepath.Join(dir2, "b.go")
	if err := os.WriteFile(f1, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := CollectTargets([]string{dir1, dir2})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("CollectTargets = %v, want 2 files", got)
	}
	if got[0] != f1 || got[1] != f2 {
		t.Errorf("CollectTargets = %v, want [%q %q]", got, f1, f2)
	}
}

func TestCollectTargetsDeduplicates(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures")
	cat := filepath.Join(dir, "cat.go")
	dog := filepath.Join(dir, "dog.go")

	// Same file passed multiple times.
	got, err := CollectTargets([]string{cat, cat, dog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("CollectTargets duplicate files = %v, want 2 files", got)
	}
	if got[0] != cat || got[1] != dog {
		t.Errorf("CollectTargets = %v, want [%q %q]", got, cat, dog)
	}

	// Directory and individual file inside that directory.
	got, err = CollectTargets([]string{dir, dog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	dogCount := 0
	for _, p := range got {
		if p == dog {
			dogCount++
		}
	}
	if dogCount != 1 {
		t.Errorf("dog.go appeared %d times, want 1", dogCount)
	}

	// Relative path variation of same file.
	relDog := filepath.Join(".", "testdata", "fixtures", "dog.go")
	got, err = CollectTargets([]string{dog, relDog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("CollectTargets with equivalent paths = %v, want 1 file", got)
	}
}

func TestCollectTargetsPropagatesError(t *testing.T) {
	dir := filepath.Join("testdata", "fixtures")
	cat := filepath.Join(dir, "cat.go")

	if _, err := CollectTargets([]string{"does-not-exist.go"}); err == nil {
		t.Error("expected error for nonexistent target")
	}
	if _, err := CollectTargets([]string{cat, "does-not-exist.go"}); err == nil {
		t.Error("expected error when second target does not exist")
	}
}
