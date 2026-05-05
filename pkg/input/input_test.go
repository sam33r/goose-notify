package input

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveArgWins(t *testing.T) {
	got, err := Resolve("from-arg", strings.NewReader("from-stdin\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-arg" {
		t.Errorf("got %q; want %q", got, "from-arg")
	}
}

func TestResolveStdinFallback(t *testing.T) {
	got, err := Resolve("", strings.NewReader("hello stdin\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello stdin" {
		t.Errorf("got %q; want %q (trailing newline trimmed)", got, "hello stdin")
	}
}

func TestResolveStdinPreservesInternalNewlines(t *testing.T) {
	got, err := Resolve("", strings.NewReader("line1\nline2\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "line1\nline2" {
		t.Errorf("got %q; want %q (only trailing newline trimmed)", got, "line1\nline2")
	}
}

func TestResolveEmptyArgEmptyStdin(t *testing.T) {
	_, err := Resolve("", strings.NewReader(""))
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("got err %v; want ErrEmpty", err)
	}
}

func TestResolveEmptyArgWhitespaceStdin(t *testing.T) {
	_, err := Resolve("", strings.NewReader("   \n\n"))
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("got err %v; want ErrEmpty (whitespace-only is empty)", err)
	}
}
