# rate-limit-fmt

Rate limits get written down in two different shapes depending on who's
reading them. A human writing a config file wants `100/min`. A service
that stores that config as JSON and hands it to other services wants
`{"limit":100,"window_seconds":60}`. Every project that has both ends up
writing this conversion once, badly, with the plural/singular unit
handling as an afterthought. This is that conversion, done once.

`ratefmt` converts between:

- **shorthand**: `<count>/<period>`, e.g. `10/s`, `100/min`, `50/10s`,
  `1000/day`, `10/1h30m`. Units accept abbreviations, full words, and
  plurals (`s`/`sec`/`second`/`seconds` are all the same unit), and are
  matched case-insensitively. A period can be a single unit or several
  run together, so `1h30m` and `1d12h` both parse.
- **JSON**: `{"limit": <int>, "window_seconds": <float>}`.

Both go through a `RateLimit{Count, Window}` value, so if you need a
third format later you only need to convert to/from that struct.

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/sonia-pine2/rate-limit-fmt"
)

func main() {
	// Parse a human-written shorthand string.
	rl, err := ratefmt.ParseShorthand("50/10s")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rl.Count, rl.Window) // 50 10s

	// Convert it to the JSON a config service would store.
	data, err := ratefmt.ShorthandToJSON("100/min")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data)) // {"limit":100,"window_seconds":60}

	// And back the other way.
	back, err := ratefmt.JSONToShorthand([]byte(`{"limit":5,"window_seconds":3600}`))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(back) // 5/hr
}
```

`RateLimit` also implements `json.Marshaler` / `json.Unmarshaler`
directly, so it can be embedded in a larger config struct:

```go
type ServiceConfig struct {
	Name  string            `json:"name"`
	Limit ratefmt.RateLimit `json:"limit"`
}
```

## Command line

```
go install github.com/sonia-pine2/rate-limit-fmt/cmd/ratefmt@latest

$ ratefmt 100/min
{"limit":100,"window_seconds":60}
$ ratefmt '{"limit":5,"window_seconds":3600}'
5/hr
$ printf '10/s\n50/10s\n' | ratefmt
```

Input starting with `{` is read as JSON, anything else as shorthand. With
no arguments it reads one rate limit per line from stdin. Every input is
converted even if an earlier one fails; the exit code is 1 if any failed.

## Awkward cases the parser handles on purpose

- `1/m` is one per minute; `1/ms` is one per millisecond. The unit is
  matched as a whole string after the leading digits are stripped, so
  `m` and `ms` never collide.
- `100/minute`, `100/min`, and `100/minutes` all parse to the same
  `RateLimit`.
- A window can have a multiplier: `50/10s` is 50 requests per 10
  seconds, not per second.
- Periods can be compound: `10/1h30m` is 10 requests per 90 minutes.
  `String()` doesn't reproduce the compound spelling - it renders
  whatever single unit divides the window evenly, so `1h30m` comes back
  out as `90min` - but it reparses to the same `RateLimit`.
- Units are case-insensitive: `10/SEC` and `10/sec` are the same.
- Zero and negative counts, zero-length windows, and unrecognized units
  are all rejected with an error rather than silently coerced.
- `String()` picks the largest unit that divides the window evenly, so
  a one-hour window renders as `5/hr`, not `5/60min` or `5/3600s`.

See `ratefmt_test.go` for the full table of cases this is checked
against.

## Status

Early skeleton. The shorthand grammar covers single and compound
periods (`10s`, `1h30m`) with an optional integer multiplier on each
segment - see the roadmap in the project notes for what's planned next.
