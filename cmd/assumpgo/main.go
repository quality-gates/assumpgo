// Command assumpgo is a static analysis tool that finds weak "assumptions" in
// Go code: negative nil/inequality checks and bare-variable truthiness that the
// "From assumptions to assertions" blog post argues should be replaced with
// explicit assertions (type assertions, type switches, positive comparisons).
package main

import (
	"flag"
	"fmt"
	"os"

	assumpgo "github.com/quality-gates/assumpgo"
)

const version = "0.1.0"

const (
	exitOK         = 0
	exitUsage      = 100
	exitAssumption = 110
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("assumpgo", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "pretty", "output format (pretty, xml)")
	fs.StringVar(format, "f", "pretty", "output format (pretty, xml) (shorthand)")
	exclude := fs.String("exclude", "", "comma separated list of files/directories to exclude")
	fs.StringVar(exclude, "e", "", "comma separated list of files/directories to exclude (shorthand)")
	output := fs.String("output", "", "write output to this file instead of stdout")
	fs.StringVar(output, "o", "", "write output to this file instead of stdout (shorthand)")
	showVersion := fs.Bool("version", false, "show the version")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	if *showVersion {
		fmt.Fprintln(stdout, version)
		return exitOK
	}

	fmt.Fprintf(stdout, "assumpgo analyser v%s by quality-gates\n\n", version)

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "error: missing target path")
		fs.Usage()
		return exitUsage
	}
	target := fs.Arg(0)

	excludes, err := assumpgo.CollectFromList(*exclude)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitUsage
	}

	targets, err := assumpgo.CollectGoFiles(target)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitUsage
	}

	analyser := assumpgo.NewAnalyser(assumpgo.NewDetector(), excludes)
	result, err := analyser.Analyse(targets)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitUsage
	}

	var renderer assumpgo.Output = assumpgo.PrettyOutput{}
	if *format == "xml" {
		renderer = assumpgo.XMLOutput{}
	}

	sink := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return exitUsage
		}
		defer f.Close()
		sink = f
	}

	if err := renderer.Output(sink, result); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return exitUsage
	}

	if result.AssumptionsCount() > 0 {
		return exitAssumption
	}

	return exitOK
}
