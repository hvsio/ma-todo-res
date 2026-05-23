package poller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hvsio/ma-todo-res/instagram"
)

// Handler is called once for each new post detected.
// Implementations must be safe to call concurrently.
type Handler func(post instagram.Post)

// Poller periodically fetches recent posts for a hashtag and calls the handler
// for posts newer than the last seen timestamp. State is kept in-memory only.
type Poller struct {
	client    *instagram.Client
	hashtag   string
	interval  time.Duration
	handler   Handler
	hashtagID string
	// lastSeen is the timestamp of the most-recently processed post.
	// Zero value means "process nothing on first tick, just establish the cursor."
	//
	// TODO: load lastSeen from persistent storage (e.g. Redis/Postgres) on
	//       startup so restarts do not re-deliver already-seen posts.
	lastSeen time.Time
}

func New(client *instagram.Client, hashtag string, interval time.Duration, handler Handler) *Poller {
	return &Poller{
		client:   client,
		hashtag:  hashtag,
		interval: interval,
		handler:  handler,
	}
}

// Run starts the polling loop and blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	id, err := p.client.HashtagID(p.hashtag)
	if err != nil {
		return fmt.Errorf("resolving hashtag %q: %w", p.hashtag, err)
	}
	p.hashtagID = id
	log.Printf("hashtag #%s resolved to node ID %s", p.hashtag, id)

	// Establish cursor on first tick without delivering posts.
	if err := p.tick(ctx, true); err != nil {
		log.Printf("initial tick error: %v", err)
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.tick(ctx, false); err != nil {
				// Log and continue — transient API errors should not crash the loop.
				log.Printf("poll error: %v", err)
			}
		}
	}
}

func (p *Poller) tick(ctx context.Context, seedOnly bool) error {
	posts, err := p.client.RecentMedia(p.hashtagID)
	if err != nil {
		return err
	}

	if len(posts) == 0 {
		return nil
	}

	// Find the newest timestamp in this batch to advance the cursor.
	newest := p.lastSeen
	for _, post := range posts {
		if post.Timestamp.After(newest) {
			newest = post.Timestamp
		}
	}

	if seedOnly {
		// First run: just record the high-water mark, don't deliver.
		p.lastSeen = newest
		log.Printf("cursor seeded at %s (%d posts visible)", newest.Format(time.RFC3339), len(posts))
		return nil
	}

	newCount := 0
	for _, post := range posts {
		if post.Timestamp.After(p.lastSeen) {
			p.handler(post)
			newCount++
		}
	}

	if newCount > 0 {
		log.Printf("delivered %d new post(s) for #%s", newCount, p.hashtag)
		// TODO: persist newest to storage here so the cursor survives restarts.
		p.lastSeen = newest
	}

	return nil
}
