// Package config parses goose-notify's CLI flags.
package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type Config struct {
	Title    string
	Duration time.Duration
	FadeIn   time.Duration
	FadeOut  time.Duration
	MaxWidth int
	OffsetY  int
}

// Parse parses argv (without the program name) and returns the config plus
// the optional positional message. If the message is missing, msg == "" and
// callers should fall back to stdin.
func Parse(args []string) (Config, string, error) {
	fs := flag.NewFlagSet("goose-notify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := Config{
		Duration: 2 * time.Second,
		FadeIn:   200 * time.Millisecond,
		FadeOut:  300 * time.Millisecond,
		MaxWidth: 600,
		OffsetY:  80,
	}
	fs.StringVar(&cfg.Title, "title", "", "bold first line above body")
	fs.DurationVar(&cfg.Duration, "duration", cfg.Duration, "how long the toast stays fully visible")
	fs.DurationVar(&cfg.FadeIn, "fade-in", cfg.FadeIn, "fade-in animation duration")
	fs.DurationVar(&cfg.FadeOut, "fade-out", cfg.FadeOut, "fade-out animation duration")
	fs.IntVar(&cfg.MaxWidth, "max-width", cfg.MaxWidth, "soft cap on toast width in pixels")
	fs.IntVar(&cfg.OffsetY, "offset-y", cfg.OffsetY, "vertical offset from top of active screen (px)")

	if err := fs.Parse(args); err != nil {
		return Config{}, "", err
	}

	rest := fs.Args()
	switch len(rest) {
	case 0:
		return cfg, "", nil
	case 1:
		return cfg, rest[0], nil
	default:
		return Config{}, "", fmt.Errorf("expected at most one positional message argument, got %d", len(rest))
	}
}
