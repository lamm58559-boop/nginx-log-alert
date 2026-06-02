package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lamm58559-boop/nginx-log-alert/config"
	"github.com/lamm58559-boop/nginx-log-alert/monitor"
	"github.com/lamm58559-boop/nginx-log-alert/notifier"
)

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %%v", err)
	}

	httpNotifier := &notifier.HTTPNotifier{
		WebhookURL: cfg.WebhookURL,
	}

	limiter := notifier.NewAlertLimiter(
		cfg.RateLimit.Enabled,
		cfg.RateLimit.MaxAlertsPerWindow,
		cfg.RateLimit.WindowDuration,
		cfg.RateLimit.AggregationEnabled,
		httpNotifier,
	)

	mon, err := monitor.NewMonitor(cfg.LogPath, cfg.Regexp, limiter)
	if err != nil {
		log.Fatalf("Failed to initialize monitor: %%v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Printf("Starting log monitor on %%s...", cfg.LogPath)
		if err := mon.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("Monitor error: %%v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal %%v, shutting down gracefully...", sig)

	cancel()

	if err := limiter.Close(); err != nil {
		log.Printf("Error closing limiter: %%v", err)
	}

	log.Println("Shutdown complete.")
}
