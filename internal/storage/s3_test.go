package storage

import "testing"

func TestBuildContentDisposition(t *testing.T) {
	if got := buildContentDisposition(""); got != "attachment" {
		t.Fatalf("expected attachment, got %q", got)
	}

	if got := buildContentDisposition("challenge.zip"); got != "attachment; filename=\"challenge.zip\"" {
		t.Fatalf("unexpected disposition: %q", got)
	}
}

func TestEscapeContentDispositionFilename(t *testing.T) {
	filename := "weird\"name\\file.zip"
	got := escapeContentDispositionFilename(filename)
	expected := "weird\\\"name\\\\file.zip"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
