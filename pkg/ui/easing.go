package ui

// EaseOutCubic maps t in [0,1] to a curve that starts fast and decelerates
// toward 1. Inputs outside [0,1] clamp to the nearest endpoint.
func EaseOutCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	u := 1 - t
	return 1 - u*u*u
}

// EaseInCubic maps t in [0,1] to a curve that starts slow and accelerates
// toward 1. Inputs outside [0,1] clamp to the nearest endpoint.
func EaseInCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * t
}

// Opacity returns the toast's current alpha (0..1) given the elapsed time
// since fade-in started and the three timeline durations. The second return
// is true once the timeline is fully complete (fade-out is done).
func Opacity(elapsedNs, fadeInNs, holdNs, fadeOutNs int64) (alpha float64, done bool) {
	if elapsedNs < fadeInNs {
		return EaseOutCubic(float64(elapsedNs) / float64(fadeInNs)), false
	}
	holdEnd := fadeInNs + holdNs
	if elapsedNs < holdEnd {
		return 1, false
	}
	fadeOutEnd := holdEnd + fadeOutNs
	if elapsedNs < fadeOutEnd {
		t := float64(elapsedNs-holdEnd) / float64(fadeOutNs)
		return 1 - EaseInCubic(t), false
	}
	return 0, true
}
