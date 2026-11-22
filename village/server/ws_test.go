package server

import "testing"

func TestSlowClientIsDropped(t *testing.T) {
	h := newHub()
	slow := h.add()
	fast := h.add()

	// Fill only the slow client's buffer, then one broadcast overflows it.
	for i := 0; i < cap(slow); i++ {
		slow <- nil
	}
	h.broadcast(activityFrame{Type: "activity", SessionID: "x"})

	if _, ok := h.clients[slow]; ok {
		t.Fatal("slow client still registered")
	}
	if _, ok := h.clients[fast]; !ok {
		t.Fatal("fast client was dropped")
	}
	for len(slow) > 0 {
		<-slow
	}
	select {
	case _, ok := <-slow:
		if ok {
			t.Fatal("slow client channel still open")
		}
	default:
		t.Fatal("slow client channel still open")
	}
}
