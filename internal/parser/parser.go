// Package parser converts kv's compact key:value syntax into a string map.
package parser

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNoInput is returned when no key:value expressions are provided.
var ErrNoInput = errors.New("at least one key:value pair is required")

// Parse parses one or more comma-separated key:value expressions.
//
// Commas and the first colon in a pair are delimiters. A delimiter or a
// backslash can be included literally by escaping it with a backslash.
func Parse(inputs []string) (map[string]string, error) {
	if len(inputs) == 0 {
		return nil, ErrNoInput
	}

	values := make(map[string]string)
	pairNumber := 0

	for inputIndex, input := range inputs {
		pairs, err := splitUnescaped(input, ',')
		if err != nil {
			return nil, fmt.Errorf("input %d: %w", inputIndex+1, err)
		}

		for _, rawPair := range pairs {
			pairNumber++
			rawPair = strings.TrimSpace(rawPair)
			if rawPair == "" {
				return nil, fmt.Errorf("pair %d is empty", pairNumber)
			}

			rawKey, rawValue, ok := splitPair(rawPair)
			if !ok {
				return nil, fmt.Errorf("pair %d %q is missing ':'", pairNumber, rawPair)
			}

			key, err := unescape(strings.TrimSpace(rawKey))
			if err != nil {
				return nil, fmt.Errorf("pair %d key: %w", pairNumber, err)
			}
			if key == "" {
				return nil, fmt.Errorf("pair %d has an empty key", pairNumber)
			}

			value, err := unescape(strings.TrimSpace(rawValue))
			if err != nil {
				return nil, fmt.Errorf("pair %d value: %w", pairNumber, err)
			}

			if _, exists := values[key]; exists {
				return nil, fmt.Errorf("pair %d duplicates key %q", pairNumber, key)
			}
			values[key] = value
		}
	}

	return values, nil
}

func splitUnescaped(input string, separator byte) ([]string, error) {
	var parts []string
	start := 0
	escaped := false

	for index := 0; index < len(input); index++ {
		character := input[index]
		if escaped {
			if !isEscapable(character) {
				return nil, fmt.Errorf("unsupported escape \\%c at byte %d", character, index)
			}
			escaped = false
			continue
		}

		if character == '\\' {
			escaped = true
			continue
		}
		if character == separator {
			parts = append(parts, input[start:index])
			start = index + 1
		}
	}

	if escaped {
		return nil, errors.New("unfinished escape at end of input")
	}

	return append(parts, input[start:]), nil
}

func splitPair(pair string) (string, string, bool) {
	escaped := false
	for index := 0; index < len(pair); index++ {
		character := pair[index]
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character == ':' {
			return pair[:index], pair[index+1:], true
		}
	}
	return "", "", false
}

func unescape(value string) (string, error) {
	var result strings.Builder
	result.Grow(len(value))

	for index := 0; index < len(value); index++ {
		character := value[index]
		if character != '\\' {
			result.WriteByte(character)
			continue
		}

		index++
		if index == len(value) {
			return "", errors.New("unfinished escape at end of value")
		}
		character = value[index]
		if !isEscapable(character) {
			return "", fmt.Errorf("unsupported escape \\%c", character)
		}
		result.WriteByte(character)
	}

	return result.String(), nil
}

func isEscapable(character byte) bool {
	return character == ',' || character == ':' || character == '\\'
}
