package ratefmt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// units maps every accepted period spelling, case-insensitively, to the
// duration a single unit represents. Abbreviations, full words, and
// plurals of the same unit all point at the same duration.
var units = map[string]time.Duration{
	"ms":           time.Millisecond,
	"millisecond":  time.Millisecond,
	"milliseconds": time.Millisecond,
	"s":            time.Second,
	"sec":          time.Second,
	"secs":         time.Second,
	"second":       time.Second,
	"seconds":      time.Second,
	"m":            time.Minute,
	"min":          time.Minute,
	"mins":         time.Minute,
	"minute":       time.Minute,
	"minutes":      time.Minute,
	"h":            time.Hour,
	"hr":           time.Hour,
	"hrs":          time.Hour,
	"hour":         time.Hour,
	"hours":        time.Hour,
	"d":            24 * time.Hour,
	"day":          24 * time.Hour,
	"days":         24 * time.Hour,
}

// unitSpec pairs a duration with the suffix String uses to render it.
type unitSpec struct {
	suffix string
	dur    time.Duration
}

// canonicalUnits is checked largest to smallest so String prefers "1/hr"
// over "60/min" when both describe the same window.
var canonicalUnits = []unitSpec{
	{"day", 24 * time.Hour},
	{"hr", time.Hour},
	{"min", time.Minute},
	{"s", time.Second},
	{"ms", time.Millisecond},
}

// ParseShorthand parses a compact rate expression such as "100/min" or
// "50/10s" into a RateLimit. The count must be a positive integer; the
// period is one or more (optional integer multiplier, unit) segments
// back to back, e.g. "min", "10s", or "1h30m" (s, sec, min, hr, day, ms,
// or their plurals). Whitespace around either half is ignored and units
// are matched case-insensitively.
func ParseShorthand(s string) (RateLimit, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return RateLimit{}, fmt.Errorf("ratefmt: %q is not in count/period form", s)
	}

	countPart := strings.TrimSpace(parts[0])
	periodPart := strings.TrimSpace(parts[1])

	count, err := strconv.Atoi(countPart)
	if err != nil {
		return RateLimit{}, fmt.Errorf("ratefmt: %q has a non-integer count: %w", s, err)
	}
	if count <= 0 {
		return RateLimit{}, fmt.Errorf("ratefmt: %q must have a positive count", s)
	}

	window, err := parsePeriod(periodPart)
	if err != nil {
		return RateLimit{}, fmt.Errorf("ratefmt: %q has an invalid period: %w", s, err)
	}

	return RateLimit{Count: count, Window: window}, nil
}

// parsePeriod parses the part of a shorthand string after the slash. It
// accepts one segment, e.g. "s" or "10s", or several run together, e.g.
// "1h30m", summing each (multiplier, unit) pair it finds.
func parsePeriod(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("period is empty")
	}

	var total time.Duration
	i := 0
	for i < len(s) {
		digitsStart := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}

		mult := 1
		if i > digitsStart {
			n, err := strconv.Atoi(s[digitsStart:i])
			if err != nil {
				return 0, err
			}
			mult = n
		}
		if mult <= 0 {
			return 0, fmt.Errorf("period multiplier must be positive")
		}

		unitStart := i
		for i < len(s) && !(s[i] >= '0' && s[i] <= '9') {
			i++
		}
		if i == unitStart {
			return 0, fmt.Errorf("no unit follows the multiplier in %q", s)
		}

		unitStr := strings.ToLower(s[unitStart:i])
		base, ok := units[unitStr]
		if !ok {
			return 0, fmt.Errorf("unrecognized unit %q", s[unitStart:i])
		}

		total += time.Duration(mult) * base
	}

	return total, nil
}

// String renders the RateLimit as compact shorthand, choosing the
// largest unit that divides the window evenly so the multiplier stays 1
// whenever possible (e.g. an hour prints as "1/hr", not "60/min").
func (r RateLimit) String() string {
	for _, u := range canonicalUnits {
		if r.Window <= 0 || r.Window%u.dur != 0 {
			continue
		}
		mult := r.Window / u.dur
		if mult == 1 {
			return fmt.Sprintf("%d/%s", r.Count, u.suffix)
		}
		return fmt.Sprintf("%d/%d%s", r.Count, mult, u.suffix)
	}
	return fmt.Sprintf("%d/%dns", r.Count, r.Window)
}
