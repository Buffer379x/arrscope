package app

import (
	"context"
	"net/url"
)

func (s *Security) Login(user, password, ip string) bool {
	return s.loginHash(user, password, ip) != ""
}

func (s *Store) Query(q url.Values, mode string, all bool) (Results, error) {
	return s.QueryContext(context.Background(), q, mode, all)
}

func (s *Store) Facets(service string) (map[string][]string, error) {
	return s.FacetsContext(context.Background(), service)
}

func (s *Store) Statistics() (Statistics, error) { return s.StatisticsContext(context.Background()) }

func (s *Store) desiredTags(ctx context.Context, service string, g TagGroups) (map[int]map[string]bool, error) {
	snap, e := readTagSnapshot(ctx, s.DB, service, g)
	return snap.desired, e
}

// replaceSyncSnapshot takes ownership of a completed upstream snapshot. Drop
// records as they are written so the growing SQLite transaction does not retain
// a second full library in Go memory. Rollback still preserves the old cache.
func (s *Store) replaceSyncSnapshot(service string, files []File, status SyncStatus) error {
	return s.replace(service, files, status, true, nil)
}

func (s *Store) replace(service string, files []File, status SyncStatus, consume bool, scope *DeltaScope) error {
	return s.replaceContext(context.Background(), service, files, status, consume, scope, nil)
}
