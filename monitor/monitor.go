package monitor

import (
	"bufio"
	"context"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/lamm58559-boop/nginx-log-alert/notifier"
)

type Monitor struct {
	logPath string
	re      *regexp.Regexp
	limiter *notifier.AlertLimiter
}

func NewMonitor(logPath string, pattern string, limiter *notifier.AlertLimiter) (*Monitor, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &Monitor{
		logPath: logPath,
		re:      re,
		limiter: limiter,
	}, nil
}

func (m *Monitor) Start(ctx context.Context) error {
	file, err := os.Open(m.logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}

			if m.re.MatchString(line) {
				_ = m.limiter.Send(ctx, line)
			}
		}
	}
}
