// Package input resolves the toast message from either a positional CLI
// argument (preferred) or stdin (fallback).
package input

import (
	"errors"
	"io"
	"strings"
)

var ErrEmpty = errors.New("empty message")

// Resolve returns the toast message. If arg is non-empty it wins; otherwise
// the entire stdin is read and its trailing newline is trimmed. Returns
// ErrEmpty if both are empty (after trim).
func Resolve(arg string, stdin io.Reader) (string, error) {
	if arg != "" {
		return arg, nil
	}
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(string(b), "\n")
	if strings.TrimSpace(s) == "" {
		return "", ErrEmpty
	}
	return s, nil
}
