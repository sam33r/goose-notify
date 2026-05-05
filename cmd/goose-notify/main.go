// Command goose-notify displays a black, fading notification toast at the
// top-center of the active macOS screen, then exits.
package main

import (
	"errors"
	"fmt"
	"os"

	"gioui.org/app"

	"github.com/sam33r/goose-notify/pkg/config"
	"github.com/sam33r/goose-notify/pkg/input"
	"github.com/sam33r/goose-notify/pkg/ui"
)

func main() {
	cfg, argMsg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
		os.Exit(1)
	}

	msg, err := input.Resolve(argMsg, os.Stdin)
	if err != nil {
		if errors.Is(err, input.ErrEmpty) {
			fmt.Fprintln(os.Stderr, "goose-notify: empty message")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "goose-notify: read input: %v\n", err)
		os.Exit(1)
	}

	t := ui.Toast{
		Title:    cfg.Title,
		Body:     msg,
		Duration: cfg.Duration,
		FadeIn:   cfg.FadeIn,
		FadeOut:  cfg.FadeOut,
		MaxWidth: cfg.MaxWidth,
		OffsetY:  cfg.OffsetY,
	}

	go func() {
		if err := ui.Run(t); err != nil {
			fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
