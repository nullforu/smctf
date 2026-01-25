package service

import "testing"

func TestTrimTo(t *testing.T) {
	if got := trimTo("short", 10); got != "short" {
		t.Fatalf("unexpected: %s", got)
	}

	if got := trimTo("toolong", 4); got != "tool" {
		t.Fatalf("unexpected: %s", got)
	}
}
