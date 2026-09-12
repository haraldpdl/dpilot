package ddev

import "testing"

func TestStatusToneAndLabelFollowDdev(t *testing.T) {
	cases := []struct {
		status ProjectStatus
		tone   Tone
		label  string
	}{
		{StatusRunning, ToneGood, "OK"}, // ddev renders a running project as green OK
		{StatusStopped, ToneBad, "stopped"},
		{StatusMissing, ToneBad, "missing"},
		{StatusPaused, ToneWarn, "paused"},
		{"unhealthy", ToneBad, "unhealthy"},
		{"something-new", ToneGood, "something-new"}, // ddev's default branch is green
	}
	for _, c := range cases {
		if got := c.status.Tone(); got != c.tone {
			t.Errorf("%q tone: got %v want %v", c.status, got, c.tone)
		}
		if got := c.status.Label(); got != c.label {
			t.Errorf("%q label: got %q want %q", c.status, got, c.label)
		}
	}
}
