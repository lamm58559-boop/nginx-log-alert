package notifier

import (
	"context"
	"sync"
	"testing"
	"time"
)

type MockNotifier struct {
	mu       sync.Mutex
	messages []string
}

func (m *MockNotifier) Send(ctx context.Context, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockNotifier) Messages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]string, len(m.messages))
	copy(res, m.messages)
	return res
}

func TestRateLimiter_UnderLimit(t *testing.T) {
	mock := &MockNotifier{}
	limiter := &AlertLimiter{
		enabled:            true,
		limit:              5,
		window:             100 * time.Millisecond,
		aggregationEnabled: true,
		notifier:           mock,
	}
	defer limiter.Close()

	for i := 0; i < 3; i++ {
		err := limiter.Send(context.Background(), "alert")
		if err != nil {
			t.Fatalf("unexpected error: %%v", err)
		}
	}

	msgs := mock.Messages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %%d", len(msgs))
	}
}

func TestRateLimiter_OverLimit(t *testing.T) {
	mock := &MockNotifier{}
	limiter := &AlertLimiter{
		enabled:            true,
		limit:              5,
		window:             100 * time.Millisecond,
		aggregationEnabled: true,
		notifier:           mock,
	}

	for i := 0; i < 10; i++ {
		err := limiter.Send(context.Background(), "alert")
		if err != nil {
			t.Fatalf("unexpected error: %%v", err)
		}
	}

	msgs := mock.Messages()
	if len(msgs) != 5 {
		t.Errorf("expected 5 messages immediately, got %%d", len(msgs))
	}

	time.Sleep(150 * time.Millisecond)

	msgs = mock.Messages()
	if len(msgs) != 6 {
		t.Errorf("expected 6 messages after window, got %%d", len(msgs))
	}

	expectedSummary := "Suppressed 5 additional alerts in the last 100ms"
	if msgs[5] != expectedSummary {
		t.Errorf("expected summary %%q, got %%q", expectedSummary, msgs[5])
	}

	limiter.Close()
}

func TestRateLimiter_Disabled(t *testing.T) {
	mock := &MockNotifier{}
	limiter := &AlertLimiter{
		enabled:            false,
		limit:              5,
		window:             100 * time.Millisecond,
		aggregationEnabled: true,
		notifier:           mock,
	}
	defer limiter.Close()

	for i := 0; i < 10; i++ {
		err := limiter.Send(context.Background(), "alert")
		if err != nil {
			t.Fatalf("unexpected error: %%v", err)
		}
	}

	msgs := mock.Messages()
	if len(msgs) != 10 {
		t.Errorf("expected 10 messages, got %%d", len(msgs))
	}
}
