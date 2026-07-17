package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/PointBreaker/kvjson/internal/interactive"
)

func TestRunCompactJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"a:b, c:d"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "{\"a\":\"b\",\"c\":\"d\"}\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrettyJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--pretty", "a:b"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "{\n  \"a\": \"b\"\n}\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q, want usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.HasPrefix(stdout.String(), "kv dev") {
		t.Fatalf("stdout = %q, want development version", stdout.String())
	}
}

func TestRunInvalidInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"not-a-pair"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing ':'") {
		t.Fatalf("stderr = %q, want parse error", stderr.String())
	}
}

func TestRunInteractive(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	editor := func() (map[string]string, error) {
		return map[string]string{"name": "Alice"}, nil
	}
	code := run(nil, &stdout, &stderr, editor)
	if code != 0 {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "{\"name\":\"Alice\"}\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunInteractiveCanceled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	editor := func() (map[string]string, error) {
		return nil, interactive.ErrCanceled
	}
	code := run(nil, &stdout, &stderr, editor)
	if code != 130 {
		t.Fatalf("run() code = %d, want 130", code)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q, want both empty", stdout.String(), stderr.String())
	}
}

func TestRunInteractiveError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	editor := func() (map[string]string, error) {
		return nil, errors.New("no terminal")
	}
	code := run(nil, &stdout, &stderr, editor)
	if code != 1 || !strings.Contains(stderr.String(), "no terminal") {
		t.Fatalf("run() code = %d, stderr = %q", code, stderr.String())
	}
}
