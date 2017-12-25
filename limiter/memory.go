package limiter

import (
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/tidwall/gjson"
)

type Memory struct {
	key  string
	age  time.Duration
	seen map[string]time.Time
	mu   *sync.Mutex
}

func (m *Memory) Configure(key string, age time.Duration) error {
	m.mu = &sync.Mutex{}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.key = key
	m.age = age

	m.seen = make(map[string]time.Time)

	go m.scrub()

	return nil
}

func (m *Memory) ProcessAndRecord(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error {
	if m.key == "" {
		return f(msg, true)
	}

	value := gjson.GetBytes(msg.Data, m.key).String()

	err := f(msg, m.shouldProcess(value))
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.seen[value] = time.Now()

	return nil
}

func (m *Memory) shouldProcess(value string) bool {
	if value == "" {
		return true
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	t, found := m.seen[value]
	if !found {
		return true
	}

	if t.Before(time.Now().Add(-1 * m.age)) {
		return true
	}

	return false
}

func (m *Memory) scrubber() {
	m.mu.Lock()
	defer m.mu.Unlock()

	killtime := time.Now().Add(-1 * m.age)

	for i, t := range m.seen {
		if t.Before(killtime) {
			delete(m.seen, i)
		}
	}
}

func (m *Memory) scrub() {
	ticker := time.NewTicker(1 * time.Minute)

	for range ticker.C {
		m.scrubber()
	}
}
