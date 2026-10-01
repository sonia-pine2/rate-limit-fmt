// Command ratefmt converts rate limits between shorthand ("100/min") and
// JSON ({"limit":100,"window_seconds":60}) from the command line.
//
// Each argument is converted on its own. With no arguments, input is read
// from standard input, one rate limit per non-empty line. The direction is
// picked from the input: anything starting with '{' is treated as JSON,
// everything else as shorthand.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	ratefmt "github.com/sonia-pine2/rate-limit-fmt"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main without the os globals so tests can drive it. It returns the
// process exit code: 0 if every input converted, 1 if any failed, 2 for a
// usage problem. A bad input doesn't stop the rest from being converted,
// since this is often fed a whole file of limits and one report listing
// every bad line is more useful than stopping at the first.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		fmt.Fprint(stdout, usage)
		return 0
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") && !isNegativeShorthand(a) {
			fmt.Fprintf(stderr, "ratefmt: unknown flag %q\n\n%s", a, usage)
			return 2
		}
	}

	failed := false
	convertOne := func(in string) {
		out, err := convert(in)
		if err != nil {
			fmt.Fprintf(stderr, "ratefmt: %v\n", err)
			failed = true
			return
		}
		fmt.Fprintln(stdout, out)
	}

	if len(args) > 0 {
		for _, a := range args {
			convertOne(a)
		}
	} else {
		sc := bufio.NewScanner(stdin)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			convertOne(line)
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintf(stderr, "ratefmt: reading input: %v\n", err)
			return 1
		}
	}

	if failed {
		return 1
	}
	return 0
}

// isNegativeShorthand lets an input like "-5/s" through to the parser so
// the user gets the "must have a positive count" error rather than an
// unknown-flag message.
func isNegativeShorthand(s string) bool {
	return len(s) > 1 && s[1] >= '0' && s[1] <= '9'
}

// convert translates a single rate limit in whichever direction its form
// implies.
func convert(in string) (string, error) {
	in = strings.TrimSpace(in)
	if strings.HasPrefix(in, "{") {
		return ratefmt.JSONToShorthand([]byte(in))
	}
	out, err := ratefmt.ShorthandToJSON(in)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

const usage = `usage: ratefmt [rate-limit ...]

Converts shorthand to JSON and JSON to shorthand. With no arguments,
reads one rate limit per line from standard input.

  ratefmt 100/min
  ratefmt '{"limit":5,"window_seconds":3600}'
  echo 10/1h30m | ratefmt
`
