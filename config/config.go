package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

type RateLimitConfig struct {
	Enabled            bool          `yaml:"enabled"`
	MaxAlertsPerWindow int           `yaml:"max_alerts_per_window"`
	WindowDurationStr  string        `yaml:"window_duration"`
	WindowDuration     time.Duration `yaml:"-"`
	AggregationEnabled bool          `yaml:"aggregation_enabled"`
}

type Config struct {
	LogPath    string          `yaml:"log_path"`
	Regexp     string          `yaml:"regexp"`
	WebhookURL string          `yaml:"webhook_url"`
	RateLimit  RateLimitConfig `yaml:"rate_limit"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.RateLimit.Enabled {
		dur, err := time.ParseDuration(cfg.RateLimit.WindowDurationStr)
		if err != nil {
			return nil, fmt.Errorf("invalid window_duration: %w", err)
		}
		cfg.RateLimit.WindowDuration = dur
	}

	return &cfg, nil
}
