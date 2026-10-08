package typing

import (
	"strconv"
	"time"
)

// Length is one of the preset test durations.
type Length int

const (
	Short Length = iota
	Medium
	Long
)

// Lengths lists the presets from shortest to longest.
var Lengths = []Length{Short, Medium, Long}

// Duration returns how long a test of length l runs.
func (l Length) Duration() time.Duration {
	switch l {
	case Short:
		return 30 * time.Second
	case Long:
		return 2 * time.Minute
	default:
		return time.Minute
	}
}

// LengthOf returns the preset with duration d, or Medium if none matches.
func LengthOf(d time.Duration) Length {
	for _, l := range Lengths {
		if l.Duration() == d {
			return l
		}
	}
	return Medium
}

// String returns the preset's name.
func (l Length) String() string {
	switch l {
	case Short:
		return "short"
	case Long:
		return "long"
	default:
		return "medium"
	}
}

// Label returns the duration in compact form, e.g. "30s".
func (l Length) Label() string { return FormatLimit(l.Duration()) }

// FormatLimit renders a test duration compactly: "30s", "60s", "120s".
func FormatLimit(d time.Duration) string {
	return strconv.Itoa(int(d.Round(time.Second)/time.Second)) + "s"
}
