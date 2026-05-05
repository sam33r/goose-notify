package ui

import (
	"math"
	"testing"
)

func nearlyEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestEaseOutCubicEndpoints(t *testing.T) {
	if got := EaseOutCubic(0); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseOutCubic(0) = %v; want 0", got)
	}
	if got := EaseOutCubic(1); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseOutCubic(1) = %v; want 1", got)
	}
}

func TestEaseOutCubicMidpoint(t *testing.T) {
	got := EaseOutCubic(0.5)
	if !nearlyEqual(got, 0.875, 1e-9) {
		t.Errorf("EaseOutCubic(0.5) = %v; want 0.875", got)
	}
}

func TestEaseInCubicEndpoints(t *testing.T) {
	if got := EaseInCubic(0); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseInCubic(0) = %v; want 0", got)
	}
	if got := EaseInCubic(1); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseInCubic(1) = %v; want 1", got)
	}
}

func TestEaseInCubicMidpoint(t *testing.T) {
	got := EaseInCubic(0.5)
	if !nearlyEqual(got, 0.125, 1e-9) {
		t.Errorf("EaseInCubic(0.5) = %v; want 0.125", got)
	}
}

func TestEaseClampsOutOfRange(t *testing.T) {
	if got := EaseOutCubic(-0.5); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseOutCubic(-0.5) = %v; want 0 (clamped)", got)
	}
	if got := EaseInCubic(1.5); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseInCubic(1.5) = %v; want 1 (clamped)", got)
	}
}

func TestOpacityFadeInPhase(t *testing.T) {
	a, done := Opacity(100_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 0.875, 1e-9) || done {
		t.Errorf("Opacity fade-in midpoint = (%v, %v); want (0.875, false)", a, done)
	}
}

func TestOpacityHoldPhase(t *testing.T) {
	a, done := Opacity(500_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 1, 1e-9) || done {
		t.Errorf("Opacity hold = (%v, %v); want (1, false)", a, done)
	}
}

func TestOpacityFadeOutPhase(t *testing.T) {
	a, done := Opacity(1_350_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 0.875, 1e-9) || done {
		t.Errorf("Opacity fade-out midpoint = (%v, %v); want (0.875, false)", a, done)
	}
}

func TestOpacityDone(t *testing.T) {
	a, done := Opacity(2_000_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 0, 1e-9) || !done {
		t.Errorf("Opacity done = (%v, %v); want (0, true)", a, done)
	}
}
