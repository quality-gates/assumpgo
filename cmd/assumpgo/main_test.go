package main

import (
	"encoding/xml"
	"fmt"
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
	// cat.go contributes the comma-ok `if` (1 boolean expression), dog.go its
	// `if` plus the `!=` comparison (2), for 1 assumption in 3 expressions.
	if !strings.Contains(stdout, "1 out of 3 boolean expressions are assumptions (33%)") {
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
	// Each file has `if x != nil` = 2 boolean expressions (the `if` and the
	// `!=`), so 2 assumptions out of 4.
	if !strings.Contains(stdout, "2 out of 4 boolean expressions are assumptions (50%)") {
		t.Errorf("unexpected summary line:\n%s", stdout)
	}
}

func TestRecursivePatternTarget(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	rootSource := []byte("package root\n\nfunc check(value any) bool {\n\tif value != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n")
	if err := os.WriteFile(filepath.Join(root, "root.go"), rootSource, 0o644); err != nil {
		t.Fatal(err)
	}

	nestedSource := []byte("package nested\n\nfunc check(value any) bool {\n\tif value != nil {\n\t\treturn true\n\t}\n\treturn false\n}\n")
	if err := os.WriteFile(filepath.Join(nested, "nested.go"), nestedSource, 0o644); err != nil {
		t.Fatal(err)
	}

	patterns := []struct {
		name string
		path string
	}{
		{name: "dot slash", path: "." + string(filepath.Separator) + "..."},
		{name: "bare", path: "..."},
	}
	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			stdout, stderr, code := runCapture(t, pattern.path)
			if code != exitAssumption {
				t.Fatalf("exit = %d, want %d (stderr: %s, stdout: %s)", code, exitAssumption, stderr, stdout)
			}
			if stderr != "" {
				t.Errorf("unexpected stderr: %s", stderr)
			}
			if !strings.Contains(stdout, "root.go") || !strings.Contains(stdout, "nested.go") {
				t.Errorf("expected recursive target files in output, got:\n%s", stdout)
			}
			if !strings.Contains(stdout, "2 out of 4 boolean expressions are assumptions (50%)") {
				t.Errorf("unexpected summary line:\n%s", stdout)
			}
		})
	}
}

func TestMultipleTargetsDeduplicated(t *testing.T) {
	stdout, _, code := runCapture(t, fixture("dog.go"), fixture("dog.go"))
	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d", code, exitAssumption)
	}
	// dog.go has 1 assumption in 2 boolean expressions (the `if` and the `!=`).
	// If analyzed twice, the assumption count would be 2.
	if !strings.Contains(stdout, "1 out of 2 boolean expressions are assumptions (50%)") {
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
	rel := fixture("dog.go")
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatal(err)
	}

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
		{
			name:    "exclude relative, target absolute",
			exclude: rel,
			target:  abs,
		},
		{
			name:    "exclude absolute, target relative",
			exclude: abs,
			target:  rel,
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

// writeSource writes a small Go file containing an assumption and returns its
// original contents for later comparison.
func writeSource(t *testing.T, path string) []byte {
	t.Helper()
	src := []byte("package p\n\nfunc F(x *int) {\n\tif x != nil {\n\t}\n}\n")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	return src
}

// TestOutputRefusesToOverwriteAnalysedFile locks in the issue #36 fix: an
// -output path that was collected as an analysis target must be refused with a
// usage error, and the source file must be left byte-for-byte intact.
func TestOutputRefusesToOverwriteAnalysedFile(t *testing.T) {
	tests := []struct {
		name   string
		target string
		output string // relative to dir, spelled as the CLI user would
		short  bool   // use the -o shorthand instead of -output
	}{
		{name: "same file", target: "victim.go", output: "victim.go"},
		{name: "same file, dot-slash output", target: "victim.go", output: "." + string(filepath.Separator) + "victim.go"},
		{name: "same file, absolute output", target: "victim.go", output: ""},
		{name: "same file, shorthand flag", target: "victim.go", output: "victim.go", short: true},
		{name: "file inside scanned directory", target: ".", output: filepath.Join("pkg", "report.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			victim := filepath.Join(dir, "victim.go")
			src := writeSource(t, victim)

			// The directory case needs a second Go file so the output path is
			// a distinct collected target, not the target argument itself.
			if tt.name == "file inside scanned directory" {
				if err := os.Mkdir(filepath.Join(dir, "pkg"), 0o755); err != nil {
					t.Fatal(err)
				}
				writeSource(t, filepath.Join(dir, "pkg", "report.go"))
			}

			output := tt.output
			if tt.name == "same file, absolute output" {
				output = victim
			}

			flag := "-output"
			if tt.short {
				flag = "-o"
			}

			_, stderr, code := runCapture(t, flag, output, tt.target)

			if code != exitUsage {
				t.Fatalf("exit = %d, want %d", code, exitUsage)
			}
			if !strings.Contains(stderr, "refusing to overwrite analysed source file: "+output) {
				t.Errorf("stderr should name the refused path:\n%s", stderr)
			}

			got, err := os.ReadFile(victim)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(src) {
				t.Errorf("analysed source was overwritten:\n%s", got)
			}
		})
	}
}

// TestOutputToNonTargetPathStillWorks pins the positive control: writing to a
// path that was not analysed must keep working.
func TestOutputToNonTargetPathStillWorks(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.go")
	src := writeSource(t, victim)
	report := filepath.Join(dir, "report.xml")

	_, stderr, code := runCapture(t, "-format", "xml", "-output", report, victim)

	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitAssumption, stderr)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "<?xml") {
		t.Errorf("report.xml should contain the XML report:\n%s", data)
	}
	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(src) {
		t.Errorf("analysed source was overwritten:\n%s", got)
	}
}

func TestExcludeNonexistentPath(t *testing.T) {
	stdout, stderr, code := runCapture(t, "-exclude", "vendor,generated", fixture("dog.go"))
	if code != exitAssumption {
		t.Fatalf("exit = %d, want %d (stderr: %s, stdout: %s)", code, exitAssumption, stderr, stdout)
	}
	if !strings.Contains(stdout, "dog.go") {
		t.Errorf("expected dog.go in output, got:\n%s", stdout)
	}
}

func writeEmpty(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty.go")
	if err := os.WriteFile(path, []byte("package p\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestUnknownFormatIsUsageError locks in the issue #37 fix: only lowercase
// pretty and xml are accepted. Anything else — including XML, json, and "" —
// is a usage error, not a silent pretty report.
func TestUnknownFormatIsUsageError(t *testing.T) {
	src := writeEmpty(t)

	tests := []struct {
		name string
		flag string
		val  string
	}{
		{name: "XML uppercase", flag: "-format", val: "XML"},
		{name: "json", flag: "-format", val: "json"},
		{name: "empty", flag: "-format", val: ""},
		{name: "Pretty mixed case", flag: "-format", val: "Pretty"},
		{name: "XML shorthand", flag: "-f", val: "XML"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runCapture(t, tt.flag, tt.val, src)

			if code != exitUsage {
				t.Fatalf("exit = %d, want %d", code, exitUsage)
			}
			if !strings.Contains(stderr, "unknown format") {
				t.Errorf("stderr should report unknown format:\n%s", stderr)
			}
			if !strings.Contains(stderr, fmt.Sprintf("%q", tt.val)) {
				t.Errorf("stderr should quote the rejected value %q:\n%s", tt.val, stderr)
			}
			if strings.Contains(stdout, "assumpgo analyser") {
				t.Errorf("banner leaked into stdout:\n%s", stdout)
			}
			if strings.Contains(stdout, "boolean expressions") {
				t.Errorf("pretty report leaked into stdout:\n%s", stdout)
			}
			if strings.HasPrefix(strings.TrimSpace(stdout), "<?xml") {
				t.Errorf("XML report leaked into stdout:\n%s", stdout)
			}
		})
	}
}

func TestAcceptedFormats(t *testing.T) {
	src := writeEmpty(t)

	tests := []struct {
		name string
		args []string
		xml  bool
	}{
		{name: "pretty", args: []string{"-format", "pretty", src}},
		{name: "pretty shorthand", args: []string{"-f", "pretty", src}},
		{name: "xml", args: []string{"-format", "xml", src}, xml: true},
		{name: "xml shorthand", args: []string{"-f", "xml", src}, xml: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, code := runCapture(t, tt.args...)
			if code != exitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr)
			}
			if strings.Contains(stderr, "unknown format") {
				t.Errorf("accepted format rejected:\n%s", stderr)
			}
			if tt.xml {
				if !strings.HasPrefix(strings.TrimSpace(stdout), "<?xml") {
					t.Errorf("xml format should emit XML:\n%s", stdout)
				}
				if strings.Contains(stdout, "assumpgo analyser") {
					t.Errorf("banner leaked into XML stdout:\n%s", stdout)
				}
				return
			}
			if !strings.Contains(stdout, "assumpgo analyser v"+version) {
				t.Errorf("pretty format should show the banner:\n%s", stdout)
			}
		})
	}
}
