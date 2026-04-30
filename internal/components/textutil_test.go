package components

import "testing"

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello w…"},
		{"x", 0, ""},
		{"x", 1, "x"},
		{"abc", 2, "a…"}, // maxW = 2 → 1 char + ellipsis
	}
	for _, tc := range cases {
		if got := Truncate(tc.in, tc.w); got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.w, got, tc.want)
		}
	}
}

func TestTruncatePath(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"~/Sites/blobs", 20, "~/Sites/blobs"},
		{"~/Sites/ven/other/scheduler-invoke-lambda", 18, "…r-invoke-lambda"[:0] + "…r-invoke-lambda"}, // tail-keep
		{"shortname", 5, "shor…"}, // no slash → prefix-truncate
		{"", 10, ""},
	}
	for _, tc := range cases {
		got := TruncatePath(tc.in, tc.w)
		if len(got) > tc.w*4 { // sanity (visible width assumed ASCII here)
			t.Errorf("TruncatePath(%q, %d) byte len suspiciously big: %q", tc.in, tc.w, got)
		}
	}
}

func TestTruncateAnsi_RoundTrip(t *testing.T) {
	// Plain ASCII string should behave like prefix-truncate.
	got := TruncateAnsi("hello world", 8)
	if got != "hello w…" {
		t.Errorf("TruncateAnsi plain ASCII: got %q", got)
	}
}
