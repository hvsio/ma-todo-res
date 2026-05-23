package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

type Config struct {
	AccessToken  string
	UserID       string
	Hashtags     []string
	PollInterval time.Duration
	GraphAPIBase string
	HTTPPort     string
}

func Load() (*Config, error) {
	token := os.Getenv("INSTAGRAM_ACCESS_TOKEN")
	userID := os.Getenv("INSTAGRAM_USER_ID")

	if token == "" {
		return nil, errors.New("INSTAGRAM_ACCESS_TOKEN is required")
	}
	if userID == "" {
		return nil, errors.New("INSTAGRAM_USER_ID is required")
	}

	var hashtags []string
	if raw := os.Getenv("INSTAGRAM_HASHTAG"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				hashtags = append(hashtags, t)
			}
		}
	}
	// Hashtags is intentionally optional — users can add them via the admin UI at runtime.

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

	port := "8080"
	if raw := os.Getenv("HTTP_PORT"); raw != "" {
		port = raw
	}

	// TODO: load long-lived token expiry and warn when close to 60-day expiration

	return &Config{
		AccessToken:  token,
		UserID:       userID,
		Hashtags:     hashtags,
		PollInterval: interval,
		GraphAPIBase: base,
		HTTPPort:     port,
	}, nil
}
