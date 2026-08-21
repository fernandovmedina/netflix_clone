package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestUnsafeRequestPathsReturnBadRequest(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := rejectUnsafePath(next)
	cases := []string{"/api/v1/stream/../secret", "/api/v1/stream/..%2fsecret", "/api/v1/stream//etc/passwd", "/api/v1/stream/x/%00", "/api/v1/stream/．．/x", "/api/v1/stream/x/..\\secret"}
	for _, target := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://stream.test/", nil)
		req.RequestURI = target
		req.URL.Path = target
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q returned %d", target, rec.Code)
		}
	}
}

func TestPathTraversalRejected(t *testing.T) {
	invalid := []string{"../", "..%2f", "/etc/passwd", "\x00", "%00", "．．", "..\\secret"}
	for _, value := range invalid {
		if _, err := uuid.Parse(value); err == nil {
			t.Errorf("asset accepted %q", value)
		}
		if qualityPattern.MatchString(value) {
			t.Errorf("quality accepted %q", value)
		}
		if segmentPattern.MatchString(value) {
			t.Errorf("segment accepted %q", value)
		}
		if validThumbnail(value) {
			t.Errorf("thumbnail accepted %q", value)
		}
	}
}
func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	for _, parts := range [][]string{{"..", "secret"}, {"/etc/passwd"}, {"hls", "..", "secret"}, {"hls", "\x00"}} {
		if path, ok := safeJoin(root, parts...); ok {
			t.Errorf("safeJoin accepted %q as %q", parts, path)
		}
	}
	want := filepath.Join(root, "hls", "asset", "master.m3u8")
	if got, ok := safeJoin(root, "hls", "asset", "master.m3u8"); !ok || got != want {
		t.Fatalf("safe path got %q,%v want %q", got, ok, want)
	}
}
