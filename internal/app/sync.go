package app

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Syncer struct {
	store      *Store
	mu         sync.Mutex
	status     map[string]SyncStatus
	cancels    map[string]context.CancelFunc
	generation map[string]int
	wg         sync.WaitGroup
	stopped    bool
	gate       chan struct{}
}

func NewSyncer(s *Store) *Syncer {
	m := &Syncer{store: s, status: map[string]SyncStatus{}, cancels: map[string]context.CancelFunc{}, generation: map[string]int{}, gate: make(chan struct{}, 1)}
	for _, service := range []string{"radarr", "sonarr"} {
		var status SyncStatus
		s.Get("sync:"+service, &status)
		if status.Running {
			status.Error = "Sync durch Neustart unterbrochen"
		}
		status.Running = false
		status.Queued = false
		m.status[service] = status
	}
	return m
}
func (m *Syncer) Statuses() map[string]SyncStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]SyncStatus{}
	for k, v := range m.status {
		out[k] = v
	}
	return out
}
func (m *Syncer) Invalidate(service string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation[service]++
	if cancel := m.cancels[service]; cancel != nil {
		cancel()
	}
	st := m.status[service]
	st.Running = false
	st.Queued = false
	m.status[service] = st
}
func (m *Syncer) Start(parent context.Context, service string, c ServiceConfig, supplemental ...ServiceConfig) error {
	if !serviceOK(service) || !c.Enabled {
		return errors.New("Service is not enabled")
	}
	m.mu.Lock()
	if m.stopped || parent.Err() != nil {
		m.mu.Unlock()
		return errors.New("App is shutting down")
	}
	if m.status[service].Running {
		m.mu.Unlock()
		return errors.New("Sync already running")
	}
	now := time.Now().UTC()
	st := m.status[service]
	st.Running = true
	st.Queued = true
	st.LastAttempt = &now
	st.Error = ""
	m.status[service] = st
	gen := m.generation[service]
	ctx, cancel := context.WithCancel(parent)
	m.cancels[service] = cancel
	m.store.Put("sync:"+service, st)
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		defer cancel()
		var files []File
		var err error
		select {
		case m.gate <- struct{}{}:
			defer func() { <-m.gate }()
			err = ctx.Err()
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err == nil {
			m.mu.Lock()
			if gen != m.generation[service] {
				m.mu.Unlock()
				return
			}
			st := m.status[service]
			st.Queued = false
			m.status[service] = st
			m.mu.Unlock()
			files, err = NewArr(c).Fetch(ctx, service)
		}
		if err == nil && len(supplemental) > 0 && supplemental[0].Enabled {
			if err = NewEmby(supplemental[0]).Enrich(ctx, service, files); err != nil {
				err = errors.New("Emby enrichment failed: " + err.Error())
			}
		}
		if err == nil {
			err = ctx.Err()
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if gen != m.generation[service] {
			return
		}
		st := m.status[service]
		st.Running = false
		st.Queued = false
		if err != nil {
			st.Error = err.Error()
			m.store.Put("sync:"+service, st)
		} else {
			now := time.Now().UTC()
			st.LastSuccess = &now
			st.Count = len(files)
			st.Error = ""
			if err = m.store.replaceSyncSnapshot(service, files, st); err != nil {
				st = m.status[service]
				st.Running = false
				st.Error = "Cache konnte nicht gespeichert werden"
				m.store.Put("sync:"+service, st)
			}
		}
		m.status[service] = st
		delete(m.cancels, service)
	}()
	return nil
}
func (m *Syncer) Shutdown() {
	m.mu.Lock()
	m.stopped = true
	for _, cancel := range m.cancels {
		cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
}
func (m *Syncer) Schedule(ctx context.Context, dispatch func()) {
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			dispatch()
		}
	}
}
