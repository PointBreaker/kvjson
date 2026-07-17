package parser

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		want   map[string]string
	}{
		{
			name:   "comma separated pairs",
			inputs: []string{"a:b, c:d"},
			want:   map[string]string{"a": "b", "c": "d"},
		},
		{
			name:   "multiple arguments",
			inputs: []string{"name:Alice", "role:admin"},
			want:   map[string]string{"name": "Alice", "role": "admin"},
		},
		{
			name:   "only first colon separates",
			inputs: []string{"url:https://example.com/a:b"},
			want:   map[string]string{"url": "https://example.com/a:b"},
		},
		{
			name:   "escaped delimiters and backslash",
			inputs: []string{`key\:part:value\,with\\slash`},
			want:   map[string]string{"key:part": `value,with\slash`},
		},
		{
			name:   "unicode",
			inputs: []string{"语言:中文, emoji:🐝"},
			want:   map[string]string{"语言": "中文", "emoji": "🐝"},
		},
		{
			name:   "empty value",
			inputs: []string{"empty:"},
			want:   map[string]string{"empty": ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.inputs)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []string
		wantErr string
	}{
		{name: "missing colon", inputs: []string{"abc"}, wantErr: "missing ':'"},
		{name: "empty key", inputs: []string{":value"}, wantErr: "empty key"},
		{name: "empty pair", inputs: []string{"a:b,,c:d"}, wantErr: "pair 2 is empty"},
		{name: "duplicate key", inputs: []string{"a:b,a:c"}, wantErr: "duplicates key"},
		{name: "trailing escape", inputs: []string{`a:value\`}, wantErr: "unfinished escape"},
		{name: "unsupported escape", inputs: []string{`a:\n`}, wantErr: "unsupported escape"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.inputs)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Parse() error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestParseNoInput(t *testing.T) {
	_, err := Parse(nil)
	if !errors.Is(err, ErrNoInput) {
		t.Fatalf("Parse() error = %v, want ErrNoInput", err)
	}
}
