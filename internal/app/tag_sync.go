package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const tagSyncSchema = `CREATE TABLE IF NOT EXISTS tag_sync_managed(service TEXT NOT NULL,endpoint TEXT NOT NULL,media_id INTEGER NOT NULL,PRIMARY KEY(service,endpoint,media_id));PRAGMA user_version=7;`

type TagSyncStatus struct {
	Running             bool       `json:"running"`
	Pending             bool       `json:"pending"`
	LastSuccess         *time.Time `json:"lastSuccess"`
	Error               string     `json:"error"`
	Changed             int        `json:"changed"`
	RetainedDefinitions int        `json:"retainedDefinitions"`
}
type TagWriter struct {
	store       *Store
	config      func() Settings
	gate        chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	events      chan string
	done        chan struct{}
	mu          sync.Mutex
	generations map[string]int
	status      map[string]TagSyncStatus
}

func NewTagWriter(ctx context.Context, store *Store, config func() Settings, gate chan struct{}) *TagWriter {
	ctx, cancel := context.WithCancel(ctx)
	w := &TagWriter{store: store, config: config, gate: gate, ctx: ctx, cancel: cancel, events: make(chan string, 2), done: make(chan struct{}), generations: map[string]int{}, status: map[string]TagSyncStatus{}}
	for _, service := range []string{"radarr", "sonarr"} {
		var retained int
		if store.Get("tagSyncRetained:"+service, &retained) == nil && retained > 0 {
			w.status[service] = TagSyncStatus{RetainedDefinitions: retained}
		}
	}
	go w.run()
	return w
}
func (w *TagWriter) Close() { w.cancel(); <-w.done }
func (w *TagWriter) Queue(service string) {
	if !serviceOK(service) || w.ctx.Err() != nil {
		return
	}
	w.mu.Lock()
	w.generations[service]++
	_ = w.store.Put("tagSyncDirty:"+service, true)
	st := w.status[service]
	st.Pending = true
	w.status[service] = st
	w.mu.Unlock()
	select {
	case w.events <- service:
	default:
	}
}
func (w *TagWriter) Statuses() map[string]TagSyncStatus {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := map[string]TagSyncStatus{}
	for k, v := range w.status {
		out[k] = v
	}
	return out
}
func (w *TagWriter) run() {
	defer close(w.done)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for _, service := range []string{"radarr", "sonarr"} {
		var dirty bool
		_ = w.store.Get("tagSyncDirty:"+service, &dirty)
		if dirty {
			w.apply(service)
		}
	}
	for {
		select {
		case <-w.ctx.Done():
			return
		case service := <-w.events:
			w.apply(service)
		case <-ticker.C:
			for _, service := range []string{"radarr", "sonarr"} {
				w.apply(service)
			}
		}
	}
}
func (w *TagWriter) apply(service string) {
	var dirty bool
	if w.store.Get("tagSyncDirty:"+service, &dirty) != nil || !dirty {
		return
	}
	ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
	defer cancel()
	select {
	case w.gate <- struct{}{}:
		defer func() { <-w.gate }()
	case <-ctx.Done():
		return
	}
	w.mu.Lock()
	gen := w.generations[service]
	st := w.status[service]
	st.Running = true
	st.Pending = true
	st.Error = ""
	w.status[service] = st
	w.mu.Unlock()
	cfg := w.config()
	c := cfg.Radarr
	if service == "sonarr" {
		c = cfg.Sonarr
	}
	unchanged := func() bool {
		fresh := w.config()
		next := fresh.Radarr
		if service == "sonarr" {
			next = fresh.Sonarr
		}
		return next == c
	}
	changed, waiting, err := w.store.ReconcileTags(ctx, service, c, unchanged)
	w.mu.Lock()
	defer w.mu.Unlock()
	st = w.status[service]
	st.Running = false
	st.Changed = changed
	_ = w.store.Get("tagSyncRetained:"+service, &st.RetainedDefinitions)
	if err != nil {
		st.Error = err.Error()
	} else if !waiting && gen == w.generations[service] {
		_ = w.store.Put("tagSyncDirty:"+service, false)
		st.Pending = false
		now := time.Now().UTC()
		st.LastSuccess = &now
	}
	w.status[service] = st
}
func tagEndpoint(c ServiceConfig) string {
	u, err := url.Parse(c.URL)
	if err != nil {
		return ""
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	sum := sha256.Sum256([]byte(u.String()))
	return hex.EncodeToString(sum[:])
}

// The upstream label restriction is a-z, digits and hyphens. Unicode is encoded
// deterministically; labels are shared/reused across traits and custom tags.
func tagLabel(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+':
			b.WriteString("-plus-")
		case r > 127:
			fmt.Fprintf(&b, "-u%x-", r)
		default:
			b.WriteByte('-')
		}
	}
	label := strings.Join(strings.FieldsFunc(b.String(), func(r rune) bool { return r == '-' }), "-")
	if len(label) > 80 {
		sum := sha256.Sum256([]byte(value))
		label = strings.TrimRight(label[:60], "-") + "-" + hex.EncodeToString(sum[:8])
	}
	return label
}

type arrTag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}
type taggedMedia struct {
	ID   int   `json:"id"`
	Tags []int `json:"tags"`
}

func (c *ArrClient) writeTags(ctx context.Context, path, method string, body, out any) error {
	if !(path == "tag" && method == "POST" || (path == "movie/editor" || path == "series/editor") && method == "PUT" || method == "DELETE" && positiveTagPath(path, "tag/")) {
		return errors.New("Tag write endpoint is not allowed")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.Base, "/")+"/api/v3/"+path, bytes.NewReader(raw))
	if err != nil {
		return errors.New("Could not create tag write request")
	}
	req.Header.Set("X-Api-Key", c.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("Tag write request failed")
	}
	defer res.Body.Close()
	if method == "DELETE" && res.StatusCode == 404 {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Tag API returned HTTP %d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(out)
	}
	_, err = io.Copy(io.Discard, io.LimitReader(res.Body, 8<<20))
	return err
}
func (s *Store) desiredTags(ctx context.Context, service string, g TagGroups) (map[int]map[string]bool, error) {
	snap, e := readTagSnapshot(ctx, s.DB, service, g)
	return snap.desired, e
}
func sameTagIDs(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[int]bool{}
	for _, id := range a {
		if id <= 0 || set[id] {
			return false
		}
		set[id] = true
	}
	for _, id := range b {
		if !set[id] {
			return false
		}
		delete(set, id)
	}
	return len(set) == 0
}

// Reconciliation is authoritative only after explicit opt-in. A durable media
// ledger permits removal on opt-out/restart without touching a different server.
func (s *Store) ReconcileTags(ctx context.Context, service string, c ServiceConfig, unchanged func() bool) (int, bool, error) {
	if !serviceOK(service) {
		return 0, false, errors.New("Invalid service")
	}
	endpoint := tagEndpoint(c)
	managed := map[int]bool{}
	rows, err := s.DB.QueryContext(ctx, "SELECT media_id FROM tag_sync_managed WHERE service=? AND endpoint=?", service, endpoint)
	if err != nil {
		return 0, false, err
	}
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, false, err
		}
		managed[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, false, err
	}
	policy, err := s.tagPolicies(ctx, service, endpoint, c.TagSync)
	if err != nil {
		return 0, false, err
	}
	var cleanupCount int
	if err = s.DB.QueryRowContext(ctx, "SELECT count(*) FROM tag_sync_cleanup WHERE service=? AND endpoint=?", service, endpoint).Scan(&cleanupCount); err != nil {
		return 0, false, err
	}
	if !c.TagSync.Active() && len(policy.remove) == 0 && cleanupCount == 0 && len(policy.keep) > 0 {
		_, err = s.DB.ExecContext(ctx, "DELETE FROM tag_sync_managed WHERE service=? AND endpoint=?", service, endpoint)
		return 0, false, err
	}
	if !c.TagSync.Active() && len(managed) == 0 && len(policy.remove) == 0 && cleanupCount == 0 {
		return 0, false, nil
	}
	if !c.Enabled {
		return 0, true, nil
	}
	if err = ValidateConfig(c); err != nil {
		return 0, false, err
	}
	if c.TagSync.Active() {
		var snapshot SyncStatus
		if s.Get("sync:"+service, &snapshot) != nil || snapshot.LastSuccess == nil || snapshot.Source != endpoint {
			return 0, true, nil
		}
	}
	snap := tagSnapshot{map[int]map[string]bool{}, map[string]map[string]bool{}, map[int]map[string]bool{}}
	desired := snap.desired
	if c.TagSync.Active() {
		snap, err = readTagSnapshot(ctx, s.DB, service, c.TagSync)
		desired = snap.desired
		if err != nil {
			return 0, false, err
		}
	}
	client := NewArr(c)
	var definitions []arrTag
	if err = client.get(ctx, "tag", nil, &definitions); err != nil {
		return 0, false, err
	}
	catalog := "movie"
	editor := "movie/editor"
	idsKey := "movieIds"
	if service == "sonarr" {
		catalog = "series"
		editor = "series/editor"
		idsKey = "seriesIds"
	}
	var media []taggedMedia
	q := url.Values{}
	if service == "radarr" {
		q.Set("excludeLocalCovers", "true")
	}
	if err = client.get(ctx, catalog, q, &media); err != nil {
		return 0, false, err
	}
	labels := map[string]int{}
	names := map[int]string{}
	for _, tag := range definitions {
		if tag.ID <= 0 || tag.Label == "" {
			return 0, false, errors.New("Invalid tag catalog")
		}
		key := strings.ToLower(tag.Label)
		names[tag.ID] = key
		if id, ok := labels[key]; !ok || tag.ID < id {
			labels[key] = tag.ID
		}
	}
	seen := map[int]bool{}
	for _, item := range media {
		if item.ID <= 0 || seen[item.ID] {
			return 0, false, errors.New("Invalid tag media catalog")
		}
		seen[item.ID] = true
		if c.TagSync.Active() || len(policy.remove) > 0 {
			managed[item.ID] = true
		}
	}
	// Persist intent before external mutations; an ambiguous response can then be
	// repaired after a retry or process restart, including an immediate opt-out.
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO tag_sync_managed VALUES(?,?,?)")
	if err != nil {
		return 0, false, err
	}
	for id := range managed {
		if _, err = stmt.ExecContext(ctx, service, endpoint, id); err != nil {
			stmt.Close()
			return 0, false, err
		}
	}
	stmt.Close()

	if c.TagSync.Active() {
		if err = s.seedTagValues(ctx, tx, service, endpoint, c.TagSync, true); err != nil {
			return 0, false, err
		}
		for field, on := range c.TagSync.fields() {
			if on {
				if _, err = tx.ExecContext(ctx, "INSERT INTO tag_sync_groups VALUES(?,?,?,'active') ON CONFLICT(service,endpoint,field) DO UPDATE SET state='active'", service, endpoint, field); err != nil {
					return 0, false, err
				}
			}
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, false, err
	}
	oldCustom := policy.custom
	policy, err = s.tagPolicies(ctx, service, endpoint, c.TagSync)
	if err != nil {
		return 0, false, err
	}
	for label := range policy.remove {
		if err = s.stageTagCleanup(ctx, service, endpoint, label); err != nil {
			return 0, false, err
		}
	}
	required := map[string]bool{}
	for id, tags := range desired {
		if !seen[id] {
			continue
		}
		for label := range tags {
			required[label] = true
		}
	}
	sorted := []string{}
	for label := range required {
		sorted = append(sorted, label)
	}
	sort.Strings(sorted)
	for _, label := range sorted {
		if labels[label] > 0 {
			continue
		}
		if !unchanged() {
			return 0, true, nil
		}
		var created arrTag
		if err = client.writeTags(ctx, "tag", "POST", map[string]string{"label": label}, &created); err != nil {
			// POST is never blindly retried: it may have committed before a timeout.
			var latest []arrTag
			if readErr := client.get(ctx, "tag", nil, &latest); readErr == nil {
				for _, tag := range latest {
					if strings.EqualFold(tag.Label, label) {
						created = tag
						break
					}
				}
			}
			if created.ID <= 0 {
				return 0, false, err
			}
		}
		if created.ID <= 0 || !strings.EqualFold(created.Label, label) {
			return 0, false, errors.New("Invalid created tag")
		}
		labels[label] = created.ID
	}

	type batch struct {
		ids, tags []int
		op        string
	}
	batches := map[string]*batch{}
	queue := func(id int, op string, tags []int) {
		if len(tags) == 0 && op != "replace" {
			return
		}
		sort.Ints(tags)
		key := op + fmt.Sprint(tags)
		if batches[key] == nil {
			batches[key] = &batch{tags: tags, op: op}
		}
		batches[key].ids = append(batches[key].ids, id)
	}
	for _, item := range media {
		if !managed[item.ID] {
			continue
		}
		want := map[int]bool{}
		for label := range desired[item.ID] {
			want[labels[label]] = true
		}
		remove := []int{}
		for _, id := range item.Tags {
			label := names[id]
			if policy.keep[label] {
				want[id] = true
				continue
			}
			if policy.remove[label] && !desired[item.ID][label] || c.TagSync.Custom && oldCustom[item.ID][label] && !desired[item.ID][label] {
				remove = append(remove, id)
			}
		}
		if tagMode(c) == "replace" {
			ids := []int{}
			for id := range want {
				ids = append(ids, id)
			}
			if sameTagIDs(ids, item.Tags) {
				continue
			}
			for _, id := range item.Tags {
				if !want[id] {
					if err = s.stageTagCleanup(ctx, service, endpoint, names[id]); err != nil {
						return 0, false, err
					}
				}
			}
			queue(item.ID, "replace", ids)
		} else {
			current := map[int]bool{}
			for _, id := range item.Tags {
				current[id] = true
			}
			for _, id := range remove {
				if err = s.stageTagCleanup(ctx, service, endpoint, names[id]); err != nil {
					return 0, false, err
				}
			}
			queue(item.ID, "remove", remove)
			add := []int{}
			for id := range want {
				if !current[id] {
					add = append(add, id)
				}
			}
			queue(item.ID, "add", add)
		}
	}
	changed := 0
	keys := []string{}
	for key := range batches {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b := batches[key]
		for start := 0; start < len(b.ids); start += 100 {
			end := start + 100
			if end > len(b.ids) {
				end = len(b.ids)
			}
			if !unchanged() {
				return changed, true, nil
			}
			var applied []taggedMedia
			if err = client.writeTags(ctx, editor, "PUT", map[string]any{idsKey: b.ids[start:end], "tags": b.tags, "applyTags": b.op, "moveFiles": false}, &applied); err != nil {
				return changed, false, err
			}
			expected := map[int]bool{}
			for _, id := range b.ids[start:end] {
				expected[id] = true
			}
			for _, item := range applied {
				if !expected[item.ID] || !tagResponseMatches(b.op, item.Tags, b.tags) {
					return changed, false, errors.New("Tag update could not be verified")
				}
				delete(expected, item.ID)
			}
			if len(expected) > 0 {
				return changed, false, errors.New("Tag update could not be verified")
			}

			changed += end - start
		}
	}
	if !unchanged() {
		return changed, true, nil
	}

	blocked, cleanErr := s.cleanupTags(ctx, service, endpoint, client, labels, required, policy.keep, unchanged)
	if cleanErr != nil {
		return changed, false, cleanErr
	}
	if err = s.Put("tagSyncRetained:"+service, blocked); err != nil {
		return changed, false, err
	}
	// Completed group retirements and custom ownership updates are atomic. Blocked
	// definitions retain their independent cleanup intent for the next file sync.
	finish, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return changed, false, e
	}
	defer finish.Rollback()
	for field, state := range policy.states {
		if state == "delete" {
			if _, e = finish.ExecContext(ctx, "DELETE FROM tag_sync_values WHERE service=? AND endpoint=? AND field=?", service, endpoint, field); e != nil {
				return changed, false, e
			}
			if _, e = finish.ExecContext(ctx, "DELETE FROM tag_sync_groups WHERE service=? AND endpoint=? AND field=?", service, endpoint, field); e != nil {
				return changed, false, e
			}
		}
	}
	if c.TagSync.Custom {
		if _, e = finish.ExecContext(ctx, "DELETE FROM tag_sync_values WHERE service=? AND endpoint=? AND field='custom' AND media_id>0", service, endpoint); e != nil {
			return changed, false, e
		}
		for id, vals := range snap.custom {
			for label := range vals {
				if _, e = finish.ExecContext(ctx, "INSERT OR IGNORE INTO tag_sync_values VALUES(?,?,'custom',?,?)", service, endpoint, label, id); e != nil {
					return changed, false, e
				}
			}
		}
	}
	if e = finish.Commit(); e != nil {
		return changed, false, e
	}
	if !c.TagSync.Active() {
		_, err = s.DB.ExecContext(ctx, "DELETE FROM tag_sync_managed WHERE service=? AND endpoint=?", service, endpoint)
	} else {
		// Deleted upstream titles no longer need a local cleanup intent.
		tx, e := s.DB.BeginTx(ctx, nil)
		if e != nil {
			return changed, false, e
		}
		defer tx.Rollback()
		for id := range managed {
			if !seen[id] {
				if _, e = tx.ExecContext(ctx, "DELETE FROM tag_sync_managed WHERE service=? AND endpoint=? AND media_id=?", service, endpoint, id); e != nil {
					return changed, false, e
				}
			}
		}
		err = tx.Commit()
	}
	return changed, false, err
}
