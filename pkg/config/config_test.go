package config

import (
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	cfg, msg, err := Parse([]string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "hello" {
		t.Errorf("msg = %q; want %q", msg, "hello")
	}
	if cfg.Title != "" {
		t.Errorf("Title = %q; want empty", cfg.Title)
	}
	if cfg.Duration != 2*time.Second {
		t.Errorf("Duration = %v; want 2s", cfg.Duration)
	}
	if cfg.FadeIn != 200*time.Millisecond {
		t.Errorf("FadeIn = %v; want 200ms", cfg.FadeIn)
	}
	if cfg.FadeOut != 300*time.Millisecond {
		t.Errorf("FadeOut = %v; want 300ms", cfg.FadeOut)
	}
	if cfg.MaxWidth != 600 {
		t.Errorf("MaxWidth = %d; want 600", cfg.MaxWidth)
	}
	if cfg.OffsetY != 80 {
		t.Errorf("OffsetY = %d; want 80", cfg.OffsetY)
	}
}

func TestParseFlags(t *testing.T) {
	cfg, msg, err := Parse([]string{
		"--title", "Status",
		"--duration", "3s",
		"--fade-in", "100ms",
		"--fade-out", "150ms",
		"--max-width", "800",
		"--offset-y", "50",
		"hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "hello world" {
		t.Errorf("msg = %q; want %q", msg, "hello world")
	}
	if cfg.Title != "Status" {
		t.Errorf("Title = %q; want %q", cfg.Title, "Status")
	}
	if cfg.Duration != 3*time.Second {
		t.Errorf("Duration = %v; want 3s", cfg.Duration)
	}
	if cfg.FadeIn != 100*time.Millisecond {
		t.Errorf("FadeIn = %v; want 100ms", cfg.FadeIn)
	}
	if cfg.FadeOut != 150*time.Millisecond {
		t.Errorf("FadeOut = %v; want 150ms", cfg.FadeOut)
	}
	if cfg.MaxWidth != 800 {
		t.Errorf("MaxWidth = %d; want 800", cfg.MaxWidth)
	}
	if cfg.OffsetY != 50 {
		t.Errorf("OffsetY = %d; want 50", cfg.OffsetY)
	}
}

func TestParseNoMessage(t *testing.T) {
	_, msg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "" {
		t.Errorf("msg = %q; want empty (so caller can fall back to stdin)", msg)
	}
}

func TestParseRejectsExtraArgs(t *testing.T) {
	_, _, err := Parse([]string{"hello", "world"})
	if err == nil {
		t.Fatal("expected error for multiple positional args, got nil")
	}
}

func TestParseBadDuration(t *testing.T) {
	_, _, err := Parse([]string{"--duration", "two seconds", "hello"})
	if err == nil {
		t.Fatal("expected error for bad duration, got nil")
	}
}
