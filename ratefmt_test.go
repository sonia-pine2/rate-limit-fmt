package ratefmt

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseShorthandValid(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want RateLimit
	}{
		{"seconds abbreviation", "10/s", RateLimit{10, time.Second}},
		{"minute abbreviation word", "100/min", RateLimit{100, time.Minute}},
		{"minute full word", "100/minute", RateLimit{100, time.Minute}},
		{"day full word", "1000/day", RateLimit{1000, 24 * time.Hour}},
		{"multiplier window", "50/10s", RateLimit{50, 10 * time.Second}},
		{"milliseconds distinct from minutes", "1/ms", RateLimit{1, time.Millisecond}},
		{"single letter minute distinct from ms", "1/m", RateLimit{1, time.Minute}},
		{"uppercase unit", "10/SEC", RateLimit{10, time.Second}},
		{"mixed case unit", "10/Min", RateLimit{10, time.Minute}},
		{"surrounding whitespace", " 10 / s ", RateLimit{10, time.Second}},
		{"plural seconds", "10/seconds", RateLimit{10, time.Second}},
		{"hour abbreviation", "5/hr", RateLimit{5, time.Hour}},
		{"compound hour and minutes", "10/1h30m", RateLimit{10, time.Hour + 30*time.Minute}},
		{"compound day and hours", "1/1d12h", RateLimit{1, 36 * time.Hour}},
		{"compound with multi-digit minutes", "5/2h45min", RateLimit{5, 2*time.Hour + 45*time.Minute}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseShorthand(tc.in)
			if err != nil {
				t.Fatalf("ParseShorthand(%q) returned error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseShorthand(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseShorthandInvalid(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"no slash", "10s"},
		{"missing count", "/s"},
		{"missing period", "10/"},
		{"zero count", "0/s"},
		{"negative count", "-5/s"},
		{"non-integer count", "ten/s"},
		{"unknown unit", "10/fortnight"},
		{"zero multiplier", "10/0s"},
		{"too many slashes", "10/5/s"},
		{"empty string", ""},
		{"only a slash", "/"},
		{"trailing multiplier with no unit", "10/1h30"},
		{"compound with unknown second unit", "10/1h30fortnight"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseShorthand(tc.in); err == nil {
				t.Fatalf("ParseShorthand(%q) succeeded, want error", tc.in)
			}
		})
	}
}

func TestRateLimitStringRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		in   RateLimit
		want string
	}{
		{"whole seconds", RateLimit{10, time.Second}, "10/s"},
		{"whole minute", RateLimit{100, time.Minute}, "100/min"},
		{"multiplier seconds", RateLimit{50, 10 * time.Second}, "50/10s"},
		{"whole day", RateLimit{1000, 24 * time.Hour}, "1000/day"},
		{"milliseconds", RateLimit{1, time.Millisecond}, "1/ms"},
		{"hour prefers hr over 60 min", RateLimit{5, time.Hour}, "5/hr"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}

			reparsed, err := ParseShorthand(tc.want)
			if err != nil {
				t.Fatalf("re-parsing %q failed: %v", tc.want, err)
			}
			if reparsed != tc.in {
				t.Fatalf("round trip mismatch: got %+v, want %+v", reparsed, tc.in)
			}
		})
	}
}

// Compound windows like "1h30m" don't have a canonical single-unit
// shorthand of their own, so String() renders them in whichever unit
// divides evenly (here, minutes) rather than reproducing "1h30m". The
// round trip is checked by value, not by exact string.
func TestParseShorthandCompoundRoundTrip(t *testing.T) {
	cases := []string{"10/1h30m", "1/1d12h", "5/2h45min"}

	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			rl, err := ParseShorthand(in)
			if err != nil {
				t.Fatalf("ParseShorthand(%q) returned error: %v", in, err)
			}

			rendered := rl.String()
			reparsed, err := ParseShorthand(rendered)
			if err != nil {
				t.Fatalf("re-parsing %q (rendered from %q) failed: %v", rendered, in, err)
			}
			if reparsed != rl {
				t.Fatalf("round trip mismatch: got %+v, want %+v", reparsed, rl)
			}
		})
	}
}

func TestShorthandToJSONAndBack(t *testing.T) {
	cases := []struct {
		name      string
		shorthand string
		wantJSON  string
	}{
		{"simple", "10/s", `{"limit":10,"window_seconds":1}`},
		{"minute", "100/min", `{"limit":100,"window_seconds":60}`},
		{"multiplier window", "50/10s", `{"limit":50,"window_seconds":10}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotJSON, err := ShorthandToJSON(tc.shorthand)
			if err != nil {
				t.Fatalf("ShorthandToJSON(%q) returned error: %v", tc.shorthand, err)
			}
			if string(gotJSON) != tc.wantJSON {
				t.Fatalf("ShorthandToJSON(%q) = %s, want %s", tc.shorthand, gotJSON, tc.wantJSON)
			}

			back, err := JSONToShorthand(gotJSON)
			if err != nil {
				t.Fatalf("JSONToShorthand(%s) returned error: %v", gotJSON, err)
			}
			if back != tc.shorthand {
				t.Fatalf("JSONToShorthand round trip = %q, want %q", back, tc.shorthand)
			}
		})
	}
}

func TestUnmarshalJSONRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"zero limit", `{"limit":0,"window_seconds":60}`},
		{"negative window", `{"limit":10,"window_seconds":-5}`},
		{"not an object", `"10/s"`},
		{"missing fields", `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rl RateLimit
			if err := json.Unmarshal([]byte(tc.in), &rl); err == nil {
				t.Fatalf("Unmarshal(%s) succeeded, want error", tc.in)
			}
		})
	}
}
