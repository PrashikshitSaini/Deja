package store

import (
	"os"
	"strings"
	"sync"

	"github.com/PrashikshitSaini/Deja/internal/matcher"
	"github.com/PrashikshitSaini/Deja/internal/model"
)

// Searcher retains a family index across queries in one shell session. It holds
// no on-disk copy of history and rebuilds after appends, replacements or purges.
type Searcher struct {
	mu       sync.Mutex
	store    Store
	info     os.FileInfo
	families map[string][]model.Event
}

func NewSearcher(path string) *Searcher { return &Searcher{store: New(path)} }

func (s *Searcher) SearchWithOptions(query, cwd string, options SearchOptions) ([]model.Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	family := matcher.Family(query)
	if family == "" {
		return nil, nil
	}
	resolved, err := s.store.resolved()
	if err != nil {
		return nil, err
	}
	err = resolved.withLock(false, func() error {
		info, err := os.Stat(resolved.Path)
		if os.IsNotExist(err) {
			s.info, s.families = nil, nil
			return nil
		}
		if err != nil {
			return err
		}
		if s.info != nil && os.SameFile(info, s.info) && info.Size() == s.info.Size() && info.ModTime().Equal(s.info.ModTime()) {
			return nil
		}
		events, err := resolved.loadUnlocked()
		if err != nil {
			s.info, s.families = nil, nil
			return err
		}
		families := make(map[string][]model.Event)
		for _, event := range events {
			families[event.Family] = append(families[event.Family], event)
		}
		s.info, s.families = info, families
		return nil
	})
	if err != nil {
		return nil, err
	}
	events, exact := s.families[family]
	if !exact {
		for name, entries := range s.families {
			if strings.HasPrefix(name, family) {
				events = append(events, entries...)
			}
		}
	}
	return searchEvents(events, query, cwd, options), nil
}
