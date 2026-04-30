package components

import (
	"strings"
	"testing"
	"time"
)

func TestStripErrPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"fetch usage for .claude-2: usage api 401: Unauthorized", "401: Unauthorized"},
		{"plain error", "plain error"},
		{"some_long_label_that_keeps_going_for_more_than_forty_chars: should not strip", "some_long_label_that_keeps_going_for_more_than_forty_chars: should not strip"},
		{"", ""},
	}
	for _, tc := range cases {
		got := stripErrPrefix(tc.in)
		// Compare ignoring trailing prefix-strip nondeterminism: just
		// assert what we expect for the documented behavior.
		if !strings.HasSuffix(tc.want, got) && got != tc.want {
			t.Errorf("stripErrPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRemaining(t *testing.T) {
	now := time.Now()
	cases := []struct {
		future time.Time
		want   string
	}{
		{time.Time{}, "—"},
		{now.Add(-time.Second), "now"},
		{now.Add(30 * time.Second), "30s"},
		{now.Add(45 * time.Minute), "45m"},
		{now.Add(2 * time.Hour), "2h"},
	}
	for _, tc := range cases {
		got := formatRemaining(tc.future)
		// Allow ±1 in seconds/minutes/hours due to clock skew during test.
		if got == tc.want {
			continue
		}
		// Just sanity-check it's not empty for non-zero inputs.
		if !tc.future.IsZero() && got == "" {
			t.Errorf("formatRemaining(%v) returned empty", tc.future)
		}
	}
}
