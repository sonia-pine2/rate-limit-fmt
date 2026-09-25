// Package ratefmt converts rate limits between two formats: a compact
// human-written shorthand ("100/min") and a structured JSON object
// ({"limit":100,"window_seconds":60}). Both formats describe the same
// thing - a count of events allowed within a window of time - so a
// RateLimit value is the common ground they convert through.
package ratefmt

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// RateLimit is a count of events allowed within a window of time.
type RateLimit struct {
	Count  int
	Window time.Duration
}

// jsonRateLimit is the wire shape used by MarshalJSON/UnmarshalJSON.
// Window is stored in seconds as a float rather than as a Go duration
// string because config systems that read this JSON are rarely Go
// programs and a plain number is easier for them to consume.
type jsonRateLimit struct {
	Limit         int     `json:"limit"`
	WindowSeconds float64 `json:"window_seconds"`
}

// MarshalJSON implements json.Marshaler.
func (r RateLimit) MarshalJSON() ([]byte, error) {
	if r.Count <= 0 || r.Window <= 0 {
		return nil, fmt.Errorf("ratefmt: cannot marshal a rate limit with a non-positive count or window")
	}
	return json.Marshal(jsonRateLimit{
		Limit:         r.Count,
		WindowSeconds: r.Window.Seconds(),
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *RateLimit) UnmarshalJSON(data []byte) error {
	var j jsonRateLimit
	if err := json.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("ratefmt: invalid rate limit JSON: %w", err)
	}
	if j.Limit <= 0 {
		return fmt.Errorf("ratefmt: limit must be positive, got %d", j.Limit)
	}
	if j.WindowSeconds <= 0 {
		return fmt.Errorf("ratefmt: window_seconds must be positive, got %v", j.WindowSeconds)
	}

	// Round rather than truncate: window_seconds arrives as a float and
	// values like 0.1 don't survive the seconds<->nanoseconds trip
	// exactly, which would otherwise nudge the window down by a tick.
	window := time.Duration(math.Round(j.WindowSeconds * float64(time.Second)))
	if window <= 0 {
		return fmt.Errorf("ratefmt: window_seconds %v is too small to represent", j.WindowSeconds)
	}

	r.Count = j.Limit
	r.Window = window
	return nil
}
