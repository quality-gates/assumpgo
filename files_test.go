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

func TestCollectGoFilesSingleFileSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "dog.go")
	file, err := filepath.Abs(filepath.Join("testdata", "fixtures", "dog.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}

	got, err := CollectGoFiles(link)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	if len(got) != 1 || got[0] != link {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", link, got, link)
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

func TestCollectGoFilesFollowsDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(target, "keep.go")
	if err := os.WriteFile(file, []byte("package target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	got, err := CollectGoFiles(link)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	want := filepath.Join(link, "keep.go")
	if len(got) != 1 || got[0] != want {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", link, got, want)
	}
}

func TestCollectGoFilesFollowsNestedDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	tree := filepath.Join(root, "tree")
	target := filepath.Join(root, "target")
	if err := os.Mkdir(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	regular := filepath.Join(tree, "regular.go")
	if err := os.WriteFile(regular, []byte("package tree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(target, "nested.go")
	if err := os.WriteFile(nested, []byte("package target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tree, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	got, err := CollectGoFiles(tree)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	want := map[string]bool{
		regular:                          false,
		filepath.Join(link, "nested.go"): false,
	}
	for _, path := range got {
		if _, ok := want[path]; ok {
			want[path] = true
		}
	}
	for path, seen := range want {
		if !seen {
			t.Errorf("expected %q to be collected; got %v", path, got)
		}
	}
}

func TestCollectGoFilesDoesNotRevisitCyclicDirectorySymlink(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "keep.go")
	if err := os.WriteFile(file, []byte("package cycle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink(dir, loop); err != nil {
		t.Fatal(err)
	}

	got, err := CollectGoFiles(dir)
	if err != nil {
		t.Fatalf("CollectGoFiles: %v", err)
	}
	if len(got) != 1 || got[0] != file {
		t.Errorf("CollectGoFiles(%q) = %v, want [%q]", dir, got, file)
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

func TestCollectFromListFollowsDirectorySymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(target, "keep.go")
	if err := os.WriteFile(file, []byte("package target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	got, err := CollectFromList(link)
	if err != nil {
		t.Fatalf("CollectFromList: %v", err)
	}
	want := filepath.Join(link, "keep.go")
	if len(got) != 1 || got[0] != want {
		t.Errorf("CollectFromList(%q) = %v, want [%q]", link, got, want)
	}
}

func TestCollectFromListIgnoresNonexistent(t *testing.T) {
	cat := filepath.Join("testdata", "fixtures", "cat.go")
	dog := filepath.Join("testdata", "fixtures", "dog.go")

	got, err := CollectFromList("vendor,generated")
	if err != nil {
		t.Fatalf("CollectFromList with nonexistent paths returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("CollectFromList with nonexistent paths = %v, want empty", got)
	}

	got, err = CollectFromList("nonexistent_before.go , " + cat + " , missing_middle , " + dog + " , nonexistent_after.go")
	if err != nil {
		t.Fatalf("CollectFromList with mixed paths returned error: %v", err)
	}
	if len(got) != 2 || got[0] != cat || got[1] != dog {
		t.Errorf("CollectFromList = %v, want [%q %q]", got, cat, dog)
	}
}

func TestCollectFromListPropagatesError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "restricted")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "file.go"), []byte("package sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(sub, 0o755)
	})

	if _, err := CollectFromList(dir); err == nil {
		t.Error("expected an error when traversing unreadable directory")
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

func TestContainsPathMatchesPathSpellings(t *testing.T) {
	dog := filepath.Join("testdata", "fixtures", "dog.go")
	absDog, err := filepath.Abs(dog)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		paths []string
		path  string
		want  bool
	}{
		{name: "exact match", paths: []string{dog}, path: dog, want: true},
		{name: "dot-slash spelling", paths: []string{dog}, path: filepath.Join(".", "testdata", "fixtures", "dog.go"), want: true},
		{name: "redundant separators", paths: []string{dog}, path: filepath.Join("testdata", "", "fixtures", "dog.go"), want: true},
		{name: "relative paths, absolute query", paths: []string{dog}, path: absDog, want: true},
		{name: "absolute paths, relative query", paths: []string{absDog}, path: dog, want: true},
		{name: "other file", paths: []string{dog}, path: filepath.Join("testdata", "fixtures", "cat.go"), want: false},
		{name: "empty paths", paths: nil, path: dog, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPath(tt.paths, tt.path); got != tt.want {
				t.Errorf("ContainsPath(%v, %q) = %v, want %v", tt.paths, tt.path, got, tt.want)
			}
		})
	}
}

func TestContainsPathMatchesFilesystemAliases(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.go")
	symlink := filepath.Join(dir, "symlink.txt")
	hardlink := filepath.Join(dir, "hardlink.txt")
	other := filepath.Join(dir, "other.go")

	if err := os.WriteFile(victim, []byte("package victim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(victim), symlink); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(victim, hardlink); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("package other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "symlink", path: symlink, want: true},
		{name: "hard link", path: hardlink, want: true},
		{name: "different file", path: other, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsPath([]string{victim}, tt.path); got != tt.want {
				t.Errorf("ContainsPath(%q, %q) = %v, want %v", victim, tt.path, got, tt.want)
			}
		})
	}
}

func TestCollectTargetsDeduplicatesAbsoluteAndRelative(t *testing.T) {
	dog := filepath.Join("testdata", "fixtures", "dog.go")
	absDog, err := filepath.Abs(dog)
	if err != nil {
		t.Fatalf("filepath.Abs(%q): %v", dog, err)
	}

	got, err := CollectTargets([]string{dog, absDog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("CollectTargets = %v, want 1 file", got)
	}
	if got[0] != dog {
		t.Errorf("CollectTargets = %v, want [%q]", got, dog)
	}

	// Reversed order keeps the first spelling given.
	got, err = CollectTargets([]string{absDog, dog})
	if err != nil {
		t.Fatalf("CollectTargets: %v", err)
	}
	if len(got) != 1 || got[0] != absDog {
		t.Errorf("CollectTargets = %v, want [%q]", got, absDog)
	}
}
