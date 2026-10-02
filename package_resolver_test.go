package assumpgo

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPackageResolverFindsConstantsDeclaredInSiblingFiles(t *testing.T) {
	dir := t.TempDir()
	defs := filepath.Join(dir, "defs.go")
	uses := filepath.Join(dir, "uses.go")
	if err := os.WriteFile(defs, []byte("package p\n\nconst Ready = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := NewPackageResolver()
	if !resolver.IsPackageConst(uses, "p", "Ready") {
		t.Error("expected sibling package constant Ready to resolve")
	}
	if resolver.IsPackageConst(uses, "p_test", "Ready") {
		t.Error("constant from package p resolved in package p_test")
	}
	if resolver.IsPackageConst(uses, "p", "Missing") {
		t.Error("unknown identifier resolved as a package constant")
	}
}

func TestPackageResolverSkipsUnreadableSiblings(t *testing.T) {
	dir := t.TempDir()
	uses := filepath.Join(dir, "uses.go")
	if err := os.WriteFile(filepath.Join(dir, "defs.go"), []byte("package p\n\nconst Ready = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing.go"), filepath.Join(dir, "dangling.go")); err != nil {
		t.Fatal(err)
	}

	resolver := NewPackageResolver()
	if !resolver.IsPackageConst(uses, "p", "Ready") {
		t.Error("expected a dangling sibling symlink to be skipped, not to stop constant indexing")
	}
}

func TestPackageResolverHonorsBuildConstraintsAndPackageNames(t *testing.T) {
	dir := t.TempDir()
	uses := filepath.Join(dir, "uses.go")
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "defs.go"), []byte("package p\n\nconst Included = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	otherGOOS := "windows"
	if runtime.GOOS == "windows" {
		otherGOOS = "linux"
	}
	platformFile := filepath.Join(dir, "defs_"+otherGOOS+".go")
	if err := os.WriteFile(platformFile, []byte("package p\n\nconst ExcludedPlatform = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_ignored.go"), []byte("package p\n\nconst ExcludedName = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte("package other\n\nconst OtherPackage = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := NewPackageResolver()
	if !resolver.IsPackageConst(uses, "p", "Included") {
		t.Error("expected a build-matched sibling constant to resolve")
	}
	for _, name := range []string{"ExcludedPlatform", "ExcludedName", "OtherPackage"} {
		if resolver.IsPackageConst(uses, "p", name) {
			t.Errorf("constant %q should not resolve in package p", name)
		}
	}
}

func TestPackageResolverUsesPhysicalDirectoryForFileSymlinks(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	linkDir := filepath.Join(root, "links")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}

	uses := filepath.Join(realDir, "uses.go")
	if err := os.WriteFile(filepath.Join(realDir, "defs.go"), []byte("package p\n\nconst Debug = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "uses.go")
	if err := os.Symlink(uses, link); err != nil {
		t.Fatal(err)
	}

	resolver := NewPackageResolver()
	if !resolver.IsPackageConst(link, "p", "Debug") {
		t.Error("expected lookup through a file symlink to use its physical package directory")
	}
}

func TestPackageResolverUsesRegisteredInMemoryDeclarations(t *testing.T) {
	dir := t.TempDir()
	defs := filepath.Join(dir, "defs.go")
	uses := filepath.Join(dir, "uses.go")
	source := "package p\n\nconst Ready = true\n"
	if err := os.WriteFile(defs, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), defs, source, 0)
	if err != nil {
		t.Fatalf("parse declarations: %v", err)
	}

	resolver := NewPackageResolver()
	resolver.addParsedFile(defs, parsed)
	if err := os.Remove(defs); err != nil {
		t.Fatal(err)
	}
	if !resolver.IsPackageConst(uses, "p", "Ready") {
		t.Error("expected resolver to use registered declarations after the source file was removed")
	}
}

func TestPackageResolverCachesDirectoryConstants(t *testing.T) {
	dir := t.TempDir()
	defs := filepath.Join(dir, "defs.go")
	uses := filepath.Join(dir, "uses.go")
	if err := os.WriteFile(defs, []byte("package p\n\nconst Ready = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uses, []byte("package p\n\nfunc check() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := NewPackageResolver()
	if !resolver.IsPackageConst(uses, "p", "Ready") {
		t.Fatal("expected the initial package constant lookup to succeed")
	}
	if err := os.WriteFile(defs, []byte("package p\n\nconst Changed = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !resolver.IsPackageConst(uses, "p", "Ready") {
		t.Error("expected the resolver to use its cached package constants")
	}
	if resolver.IsPackageConst(uses, "p", "Changed") {
		t.Error("resolver rescanned a package directory after caching it")
	}
}
