package realtime

import (
	"testing"
	"time"
)

func TestSSEHubSubscribeBroadcast(t *testing.T) {
	hub := NewSSEHub()
	ch, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()

	hub.Broadcast("hello")

	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatalf("channel closed unexpectedly")
		}

		if msg != "hello" {
			t.Fatalf("expected message %q, got %q", "hello", msg)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for broadcast")
	}
}

func TestSSEHubUnsubscribeClosesChannel(t *testing.T) {
	hub := NewSSEHub()
	ch, unsubscribe := hub.Subscribe(1)

	unsubscribe()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatalf("expected channel to be closed")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for channel close")
	}
}
