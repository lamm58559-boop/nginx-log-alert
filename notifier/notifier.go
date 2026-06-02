package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Notifier interface {
	Send(ctx context.Context, message string) error
}

type HTTPNotifier struct {
	WebhookURL string
}

func (h *HTTPNotifier) Send(ctx context.Context, message string) error {
	payload := map[string]string{"text": message}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.WebhookURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

type AlertLimiter struct {
	mu                 sync.Mutex
	enabled            bool
	limit              int
	window             time.Duration
	aggregationEnabled bool
	notifier           Notifier

	alertCount  int
	suppressed  int
	windowStart time.Time
	timer       *time.Timer
	closed      bool
}

func NewAlertLimiter(enabled bool, limit int, window time.Duration, aggregationEnabled bool, notifier Notifier) *AlertLimiter {
	return &AlertLimiter{
	enabled:            enabled,
	limit:              limit,
	window:             window,
	aggregationEnabled: aggregationEnabled,
	notifier:           notifier,
	}
}

func (al *AlertLimiter) Send(ctx context.Context, message string) error {
	al.mu.Lock()
	if !al.enabled {
		al.mu.Unlock()
		return al.notifier.Send(ctx, message)
	}

	if al.closed {
		al.mu.Unlock()
		return fmt.Errorf("limiter is closed")
	}

	now := time.Now()
	if al.windowStart.IsZero() || now.Sub(al.windowStart) >= al.window {
		al.startNewWindow(now)
	}

	if al.alertCount < al.limit {
		al.alertCount++
		al.mu.Unlock()
		return al.notifier.Send(ctx, message)
	}

	if al.aggregationEnabled {
		al.suppressed++
	}
	al.mu.Unlock()
	return nil
}

func (al *AlertLimiter) startNewWindow(now time.Time) {
	if al.timer != nil {
		al.timer.Stop()
	}
	al.windowStart = now
	al.alertCount = 0
	al.suppressed = 0

	var t *time.Timer
	t = time.AfterFunc(al.window, func() {
		al.mu.Lock()
		defer al.mu.Unlock()

		if al.closed || al.timer != t {
			return
		}

		if al.suppressed > 0 && al.aggregationEnabled {
			msg := fmt.Sprintf("Suppressed %%d additional alerts in the last %%s", al.suppressed, al.window.String())
			go func(m string) {
				_ = al.notifier.Send(context.Background(), m)
			}(msg)
		}

		al.windowStart = time.Time{}
		al.alertCount = 0
		al.suppressed = 0
		al.timer = nil
	})
	al.timer = t
}

func (al *AlertLimiter) Close() error {
	al.mu.Lock()
	if al.closed {
		al.mu.Unlock()
		return nil
	}
	al.closed = true

	if al.timer != nil {
		al.timer.Stop()
	}

	suppressed := al.suppressed
	window := al.window
	aggregationEnabled := al.aggregationEnabled
	al.mu.Unlock()

	if suppressed > 0 && aggregationEnabled {
		msg := fmt.Sprintf("Suppressed %%d additional alerts in the last %%s", suppressed, window.String())
		return al.notifier.Send(context.Background(), msg)
	}
	return nil
}
