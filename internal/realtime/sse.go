package realtime

import "sync"

type SSEHub struct {
	mu          sync.Mutex
	subscribers map[chan string]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{subscribers: make(map[chan string]struct{})}
}

func (h *SSEHub) Subscribe(buffer int) (chan string, func()) {
	if buffer <= 0 {
		buffer = 1
	}

	ch := make(chan string, buffer)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}

		h.mu.Unlock()
	}

	return ch, unsubscribe
}

func (h *SSEHub) Broadcast(payload string) {
	h.mu.Lock()
	for ch := range h.subscribers {
		select {
		case ch <- payload:
		default:
		}
	}

	h.mu.Unlock()
}
