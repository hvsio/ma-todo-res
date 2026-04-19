package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	AccessToken  string
	UserID       string
	Hashtag      string
	PollInterval time.Duration
	GraphAPIBase string
}

func Load() (*Config, error) {
	token := os.Getenv("INSTAGRAM_ACCESS_TOKEN")
	userID := os.Getenv("INSTAGRAM_USER_ID")
	hashtag := os.Getenv("INSTAGRAM_HASHTAG")

	if token == "" {
		return nil, errors.New("INSTAGRAM_ACCESS_TOKEN is required")
	}
	if userID == "" {
		return nil, errors.New("INSTAGRAM_USER_ID is required")
	}
	if hashtag == "" {
		return nil, errors.New("INSTAGRAM_HASHTAG is required")
	}

	interval := 5 * time.Minute
	if raw := os.Getenv("POLL_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, errors.New("POLL_INTERVAL must be a valid Go duration (e.g. 5m, 30s)")
		}
		interval = parsed
	}

	base := "https://graph.facebook.com/v22.0"
	if raw := os.Getenv("GRAPH_API_BASE"); raw != "" {
		base = raw
	}

	// TODO: support multiple hashtags via comma-separated INSTAGRAM_HASHTAG list
	// TODO: load long-lived token expiry and warn when close to 60-day expiration
	return &Config{
		AccessToken:  token,
		UserID:       userID,
		Hashtag:      hashtag,
		PollInterval: interval,
		GraphAPIBase: base,
	}, nil
}
