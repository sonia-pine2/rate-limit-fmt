package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		stdin      string
		wantCode   int
		wantStdout string
		wantStderr string // substring; empty means stderr must be empty
	}{
		{
			name:       "shorthand argument",
			args:       []string{"100/min"},
			wantStdout: "{\"limit\":100,\"window_seconds\":60}\n",
		},
		{
			name:       "json argument",
			args:       []string{`{"limit":5,"window_seconds":3600}`},
			wantStdout: "5/hr\n",
		},
		{
			name:       "several arguments",
			args:       []string{"10/s", "1/day"},
			wantStdout: "{\"limit\":10,\"window_seconds\":1}\n{\"limit\":1,\"window_seconds\":86400}\n",
		},
		{
			name:       "stdin lines skip blanks",
			stdin:      "10/s\n\n  50/10s  \n",
			wantStdout: "{\"limit\":10,\"window_seconds\":1}\n{\"limit\":50,\"window_seconds\":10}\n",
		},
		{
			name:       "bad input does not stop later input",
			args:       []string{"nope", "10/s"},
			wantCode:   1,
			wantStdout: "{\"limit\":10,\"window_seconds\":1}\n",
			wantStderr: "not in count/period form",
		},
		{
			name:       "negative count reaches the parser",
			args:       []string{"-5/s"},
			wantCode:   1,
			wantStderr: "positive count",
		},
		{
			name:       "unknown flag",
			args:       []string{"-x"},
			wantCode:   2,
			wantStderr: "unknown flag",
		},
		{
			name:       "help",
			args:       []string{"-h"},
			wantStdout: usage,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(tc.stdin), &stdout, &stderr)

			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tc.wantCode, stderr.String())
			}
			if stdout.String() != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
			if tc.wantStderr == "" && stderr.Len() > 0 {
				t.Errorf("unexpected stderr: %s", stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}
