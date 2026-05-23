package manager

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/hvsio/ma-todo-res/config"
	"github.com/hvsio/ma-todo-res/instagram"
	"github.com/hvsio/ma-todo-res/poller"
	"github.com/hvsio/ma-todo-res/store"
)

// Manager controls the lifecycle of per-hashtag poller goroutines.
type Manager struct {
	mu     sync.Mutex
	active map[string]context.CancelFunc
	wg     sync.WaitGroup
	root   context.Context
	cancel context.CancelFunc
	cfg    *config.Config
	store  *store.Store
}

func New(cfg *config.Config, st *store.Store) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		active: make(map[string]context.CancelFunc),
		root:   ctx,
		cancel: cancel,
		cfg:    cfg,
		store:  st,
	}
}

// Add starts a poller for the given hashtag. Returns an error if it is already running.
func (m *Manager) Add(hashtag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.active[hashtag]; exists {
		return fmt.Errorf("hashtag %q is already being monitored", hashtag)
	}

	ctx, cancel := context.WithCancel(m.root)
	m.active[hashtag] = cancel

	client := instagram.NewClient(m.cfg.GraphAPIBase, m.cfg.AccessToken, m.cfg.UserID)
	handler := func(p instagram.Post) {
		m.store.Add(store.StoredPost{Post: p, Hashtag: hashtag})
	}
	p := poller.New(client, hashtag, m.cfg.PollInterval, handler)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := p.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("poller[%s] stopped: %v", hashtag, err)
		}
	}()

	log.Printf("started poller for #%s", hashtag)
	return nil
}

// Remove stops the poller for the given hashtag. No-op if not running.
func (m *Manager) Remove(hashtag string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cancel, exists := m.active[hashtag]
	if !exists {
		return
	}
	cancel()
	delete(m.active, hashtag)
	log.Printf("stopped poller for #%s", hashtag)
}

// List returns the currently active hashtags in sorted order.
func (m *Manager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	tags := make([]string, 0, len(m.active))
	for t := range m.active {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// Shutdown cancels all active pollers and waits for their goroutines to exit.
func (m *Manager) Shutdown() {
	m.cancel()
	m.wg.Wait()
}
