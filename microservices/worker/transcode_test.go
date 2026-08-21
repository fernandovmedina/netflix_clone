package main

import (
	"strings"
	"testing"
)

func TestEvenWidth(t *testing.T) {
	tests := []struct{ w, h, target int }{{1919, 1079, 144}, {1919, 1079, 720}, {1366, 768, 480}, {1366, 768, 720}}
	for _, tt := range tests {
		got := evenWidth(tt.w, tt.h, tt.target)
		if got%2 != 0 {
			t.Errorf("evenWidth(%d,%d,%d)=%d", tt.w, tt.h, tt.target, got)
		}
		if got < 2 {
			t.Errorf("invalid width %d", got)
		}
	}
}
func TestNoAudioArgs(t *testing.T) {
	args := strings.Join(ffmpegArgs("source.mp4", "out", profileFor(360), 24, false), " ")
	for _, forbidden := range []string{"-c:a", "-b:a", "0:a"} {
		if strings.Contains(args, forbidden) {
			t.Fatalf("no-audio args contain %q: %s", forbidden, args)
		}
	}
	if !strings.Contains(args, "-an") {
		t.Fatalf("no-audio args omit -an: %s", args)
	}
}
func TestAudioArgs(t *testing.T) {
	args := strings.Join(ffmpegArgs("source.mp4", "out", profileFor(360), 24, true), " ")
	if !strings.Contains(args, "-c:a aac") || !strings.Contains(args, "-map 0:a:0") {
		t.Fatalf("audio args incomplete: %s", args)
	}
}
func TestMasterPlaylist(t *testing.T) {
	master := masterPlaylist([]variant{{Quality: "720p", Width: 1280, Height: 720, Profile: profileFor(720), Audio: false}})
	for _, want := range []string{"BANDWIDTH=2996000", "AVERAGE-BANDWIDTH=2800000", "RESOLUTION=1280x720", "CODECS=\"avc1.4d401f\"", "720p/playlist.m3u8"} {
		if !strings.Contains(master, want) {
			t.Errorf("master missing %q:\n%s", want, master)
		}
	}
}
