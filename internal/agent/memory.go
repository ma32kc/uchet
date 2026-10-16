package agent

import (
	"sync"
	"time"
)

type Memory struct {
	mu       sync.Mutex
	sessions map[int64]*session
	maxMsgs  int
	ttl      time.Duration
}

type session struct {
	messages []chatMessage
	updated  time.Time
}

func NewMemory(maxMsgs int, ttl time.Duration) *Memory {
	if maxMsgs <= 0 {
		maxMsgs = 20
	}
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	return &Memory{
		sessions: make(map[int64]*session),
		maxMsgs:  maxMsgs,
		ttl:      ttl,
	}
}

func (m *Memory) Get(chatID int64) []chatMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked()
	s, ok := m.sessions[chatID]
	if !ok {
		return nil
	}
	out := make([]chatMessage, len(s.messages))
	copy(out, s.messages)
	return out
}

func (m *Memory) Append(chatID int64, msgs ...chatMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[chatID]
	if !ok {
		s = &session{}
		m.sessions[chatID] = s
	}
	s.messages = append(s.messages, msgs...)
	if len(s.messages) > m.maxMsgs {
		s.messages = s.messages[len(s.messages)-m.maxMsgs:]
	}
	s.updated = time.Now()
}

func (m *Memory) Reset(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chatID)
}

func (m *Memory) cleanupLocked() {
	now := time.Now()
	for id, s := range m.sessions {
		if now.Sub(s.updated) > m.ttl {
			delete(m.sessions, id)
		}
	}
}
