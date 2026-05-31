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
