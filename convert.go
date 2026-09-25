package ratefmt

import "encoding/json"

// ShorthandToJSON converts a compact rate expression like "100/min" into
// its canonical JSON form, e.g. {"limit":100,"window_seconds":60}.
func ShorthandToJSON(shorthand string) ([]byte, error) {
	rl, err := ParseShorthand(shorthand)
	if err != nil {
		return nil, err
	}
	return json.Marshal(rl)
}

// JSONToShorthand converts a JSON rate limit object back into its
// shorthand form.
func JSONToShorthand(data []byte) (string, error) {
	var rl RateLimit
	if err := json.Unmarshal(data, &rl); err != nil {
		return "", err
	}
	return rl.String(), nil
}
