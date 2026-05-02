package idempotency

import "sync"

type Store struct {
	seen sync.Map
}

func New() *Store {
	return &Store{}
}

func (s *Store) MarkSeen(id string) (alreadySeen bool) {
	_, loaded := s.seen.LoadOrStore(id, struct{}{})
	return loaded
}
