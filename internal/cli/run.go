// Package cli implements kv's command-line interface.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/PointBreaker/kvjson/internal/buildinfo"
	"github.com/PointBreaker/kvjson/internal/interactive"
	"github.com/PointBreaker/kvjson/internal/parser"
)

type editorFunc func() (map[string]string, error)

// Run executes the CLI and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, interactive.Run)
}

func run(args []string, stdout, stderr io.Writer, editor editorFunc) int {
	if wantsHelp(args) {
		printUsage(stdout)
		return 0
	}

	flags := flag.NewFlagSet("kv", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { printUsage(stderr) }

	var pretty bool
	var showVersion bool
	flags.BoolVar(&pretty, "pretty", false, "pretty-print the generated JSON")
	flags.BoolVar(&pretty, "p", false, "shorthand for --pretty")
	flags.BoolVar(&showVersion, "version", false, "print version information")
	flags.BoolVar(&showVersion, "v", false, "shorthand for --version")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if showVersion {
		fmt.Fprintln(stdout, buildinfo.String())
		return 0
	}

	var values map[string]string
	var err error
	if len(flags.Args()) == 0 {
		values, err = editor()
		if errors.Is(err, interactive.ErrCanceled) {
			return 130
		}
		if err != nil {
			fmt.Fprintf(stderr, "kv: interactive mode: %v\n", err)
			return 1
		}
	} else {
		values, err = parser.Parse(flags.Args())
		if err != nil {
			fmt.Fprintf(stderr, "kv: %v\n", err)
			fmt.Fprintln(stderr, "Run 'kv --help' for usage.")
			return 2
		}
	}

	var output []byte
	if pretty {
		output, err = json.MarshalIndent(values, "", "  ")
	} else {
		output, err = json.Marshal(values)
	}
	if err != nil {
		fmt.Fprintf(stderr, "kv: encode JSON: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, string(output))
	return 0
}

func wantsHelp(args []string) bool {
	for _, argument := range args {
		if argument == "--" {
			return false
		}
		if argument == "-h" || argument == "--help" {
			return true
		}
	}
	return false
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, `kv turns compact key:value pairs into a JSON object.

Usage:
  kv
  kv [options] 'key:value, key:value'
  kv [options] 'key:value' 'key:value'

Examples:
  kv
  kv 'a:b, c:d'
  kv 'message:hello\, world, url:https://example.com/a:b'
  curl https://example.com -H 'Content-Type: application/json' --data "$(kv 'a:b, c:d')"

Escapes:
  \,  literal comma
  \:  literal colon
  \\  literal backslash

Options:
  -p, --pretty    pretty-print the generated JSON
  -v, --version   print version information
  -h, --help      show this help`)
}
