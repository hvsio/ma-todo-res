package store

import (
	"sync"

	"github.com/hvsio/ma-todo-res/instagram"
)

const DefaultCapacity = 200

// StoredPost wraps instagram.Post with the hashtag it was fetched under.
type StoredPost struct {
	instagram.Post
	Hashtag string `json:"hashtag"`
}

// Store is a fixed-capacity ring buffer, safe for concurrent use.
type Store struct {
	mu    sync.RWMutex
	buf   []StoredPost
	cap   int
	head  int // next write slot
	count int // valid entries, 0..cap
}

func New(capacity int) *Store {
	return &Store{
		buf: make([]StoredPost, capacity),
		cap: capacity,
	}
}

// Add inserts a post, overwriting the oldest entry when the buffer is full.
func (s *Store) Add(p StoredPost) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf[s.head] = p
	s.head = (s.head + 1) % s.cap
	if s.count < s.cap {
		s.count++
	}
}

// List returns a copy of all stored posts, newest-first.
func (s *Store) List() []StoredPost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]StoredPost, s.count)
	for i := 0; i < s.count; i++ {
		// Walk backwards from the last written slot.
		idx := (s.head - 1 - i + s.cap) % s.cap
		out[i] = s.buf[idx]
	}
	return out
}
