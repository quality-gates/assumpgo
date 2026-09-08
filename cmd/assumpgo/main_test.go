package main

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCapture invokes run with real temp files for stdout/stderr (run takes
// *os.File) and returns their contents plus the exit code.
func runCapture(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	dir := t.TempDir()
	outF, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	defer outF.Close()
	errF, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer errF.Close()

	code = run(args, outF, errF)

	return readFile(t, outF.Name()), readFile(t, errF.Name()), code
}

func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fixture(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}

// TestXMLStdoutIsWellFormed guards the advertised `-format xml <path>` CI form:
// stdout must be valid XML with no human banner leaking into it.
func TestXMLStdoutIsWellFormed(t *testing.T) {
	stdout, _, code := runCapture(t, "-format", "xml", fixture("dog.go"))

	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	if strings.Contains(stdout, "assumpgo analyser") {
		t.Errorf("banner leaked into XML stdout:\n%s", stdout)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "<?xml") {
		t.Errorf("XML stdout does not start with the XML prolog:\n%s", stdout)
	}

	// The whole stream must parse without error.
	dec := xml.NewDecoder(strings.NewReader(stdout))
	for {
		_, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("stdout is not well-formed XML: %v\n%s", err, stdout)
		}
	}
}

// TestPrettyKeepsBanner locks in that the human format is unchanged.
func TestPrettyKeepsBanner(t *testing.T) {
	stdout, _, code := runCapture(t, fixture("dog.go"))

	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	if !strings.Contains(stdout, "assumpgo analyser v"+version) {
		t.Errorf("pretty output should show the banner:\n%s", stdout)
	}
}

func TestVersionFlag(t *testing.T) {
	stdout, _, code := runCapture(t, "-version")
	if code != exitOK {
		t.Fatalf("exit = %d, want %d", code, exitOK)
	}
	if strings.TrimSpace(stdout) != version {
		t.Errorf("version output = %q, want %q", strings.TrimSpace(stdout), version)
	}
}

func TestMissingPathIsUsageError(t *testing.T) {
	_, stderr, code := runCapture(t)
	if code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "missing target path") {
		t.Errorf("stderr should report the missing path:\n%s", stderr)
	}
}

func TestMultipleTargetFiles(t *testing.T) {
	stdout, _, code := runCapture(t, fixture("cat.go"), fixture("dog.go"))
	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	if !strings.Contains(stdout, "dog.go") {
		t.Errorf("expected dog.go in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "1 out of 2 boolean expressions are assumptions (50%)") {
		t.Errorf("unexpected summary line:\n%s", stdout)
	}
}

func TestMultipleTargetDirectories(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	f1 := filepath.Join(dir1, "a.go")
	f2 := filepath.Join(dir2, "b.go")
	if err := os.WriteFile(f1, []byte("package a\nfunc check(x any) bool {\n\tif x != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("package b\nfunc check(y any) bool {\n\tif y != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runCapture(t, dir1, dir2)
	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	if !strings.Contains(stdout, "a.go") || !strings.Contains(stdout, "b.go") {
		t.Errorf("expected both a.go and b.go in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "2 out of 2 boolean expressions are assumptions (100%)") {
		t.Errorf("unexpected summary line:\n%s", stdout)
	}
}

func TestMultipleTargetsDeduplicated(t *testing.T) {
	stdout, _, code := runCapture(t, fixture("dog.go"), fixture("dog.go"))
	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	// dog.go has 1 assumption. If analyzed twice, count would be 2.
	if !strings.Contains(stdout, "1 out of 1 boolean expressions are assumptions (100%)") {
		t.Errorf("expected deduplicated summary of 1 expression, got:\n%s", stdout)
	}
}

func TestMultipleTargetsErrorOnMissingPath(t *testing.T) {
	_, stderr, code := runCapture(t, fixture("dog.go"), "nonexistent-file.go")
	if code != exitUsage {
		t.Fatalf("exit = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(stderr, "error:") {
		t.Errorf("expected error message on stderr, got:\n%s", stderr)
	}
}

func TestExcludePathSyntaxVariations(t *testing.T) {
	tests := []struct {
		name    string
		exclude string
		target  string
	}{
		{
			name:    "exclude with leading dot-slash",
			exclude: "." + string(filepath.Separator) + fixture("dog.go"),
			target:  fixture("dog.go"),
		},
		{
			name:    "target with leading dot-slash",
			exclude: fixture("dog.go"),
			target:  "." + string(filepath.Separator) + fixture("dog.go"),
		},
		{
			name:    "exclude with redundant separators",
			exclude: ".." + string(filepath.Separator) + ".." + string(filepath.Separator) + "testdata" + string(filepath.Separator) + string(filepath.Separator) + "fixtures" + string(filepath.Separator) + "dog.go",
			target:  fixture("dog.go"),
		},
		{
			name:    "target with redundant separators",
			exclude: fixture("dog.go"),
			target:  ".." + string(filepath.Separator) + ".." + string(filepath.Separator) + "testdata" + string(filepath.Separator) + string(filepath.Separator) + "fixtures" + string(filepath.Separator) + "dog.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runCapture(t, "-exclude", tt.exclude, tt.target)
			if code != exitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s, stdout: %s)", code, exitOK, stderr, stdout)
			}
			if strings.Contains(stdout, "dog.go |") {
				t.Errorf("dog.go was not excluded:\n%s", stdout)
			}
			if !strings.Contains(stdout, "0 out of 0 boolean expressions are assumptions (0%)") {
				t.Errorf("unexpected summary line:\n%s", stdout)
			}
		})
	}
}
