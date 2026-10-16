package agent

import (
	"testing"
	"time"
)

func TestMemoryAppendAndTrim(t *testing.T) {
	m := NewMemory(3, time.Hour)
	m.Append(1, chatMessage{Role: "user", Content: "1"})
	m.Append(1, chatMessage{Role: "user", Content: "2"})
	m.Append(1, chatMessage{Role: "user", Content: "3"})
	m.Append(1, chatMessage{Role: "user", Content: "4"})

	got := m.Get(1)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got[0].Content != "2" || got[2].Content != "4" {
		t.Fatalf("unexpected trim: %+v", got)
	}
}

func TestMemoryReset(t *testing.T) {
	m := NewMemory(10, time.Hour)
	m.Append(7, chatMessage{Role: "user", Content: "hi"})
	m.Reset(7)
	if len(m.Get(7)) != 0 {
		t.Fatal("expected empty after reset")
	}
}
