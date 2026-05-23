package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hvsio/ma-todo-res/config"
	"github.com/hvsio/ma-todo-res/instagram"
	"github.com/hvsio/ma-todo-res/poller"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	client := instagram.NewClient(cfg.GraphAPIBase, cfg.AccessToken, cfg.UserID)

	// TODO: replace this log handler with your delivery mechanism, e.g.:
	//   - HTTP POST to a webhook URL
	//   - publish to a message queue (Kafka, RabbitMQ, SQS)
	//   - write to a database
	handler := func(post instagram.Post) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(post); err != nil {
			log.Printf("failed to encode post %s: %v", post.ID, err)
		}
	}

	p := poller.New(client, cfg.Hashtag, cfg.PollInterval, handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("starting hashtag poller: #%s every %s", cfg.Hashtag, cfg.PollInterval)

	if err := p.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("poller stopped: %v", err)
	}

	log.Println("shutdown complete")
}
