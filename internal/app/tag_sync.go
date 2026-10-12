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
	Running     bool       `json:"running"`
	Pending     bool       `json:"pending"`
	LastSuccess *time.Time `json:"lastSuccess"`
	Error       string     `json:"error"`
	Changed     int        `json:"changed"`
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
	if !(path == "tag" && method == "POST" || (path == "movie/editor" || path == "series/editor") && method == "PUT") {
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
	custom := "'[]'"
	if g.Custom {
		custom = customJSON
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT f.media_id,f.resolution,f.codec,f.source,f.processing,f.hdr,f.bitDepth,f.audio,f.channels,f.language,"+custom+" FROM files f WHERE f.service=?", service)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]map[string]bool{}
	for rows.Next() {
		var id int
		var resolution, codec, source, processing, hdr, depth, audio, channels, language, customRaw string
		if err = rows.Scan(&id, &resolution, &codec, &source, &processing, &hdr, &depth, &audio, &channels, &language, &customRaw); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]bool{}
		}
		add := func(enabled bool, value, separator string) {
			if !enabled {
				return
			}
			for _, part := range strings.Split(value, separator) {
				if label := tagLabel(part); label != "" {
					out[id][label] = true
				}
			}
		}
		add(g.Resolution, resolution, " + ")
		add(g.Codec, codec, " + ")
		add(g.Source, source, " + ")
		add(g.Source, processing, " + ")
		add(g.HDR, hdr, " + ")
		add(g.BitDepth, depth, " + ")
		add(g.Audio, audio, " + ")
		add(g.Channels, channels, " + ")
		add(g.Language, language, ", ")
		if g.Custom {
			var tags []CustomTag
			if err = json.Unmarshal([]byte(customRaw), &tags); err != nil {
				return nil, err
			}
			for _, tag := range tags {
				add(true, tag.Name, "\x00")
			}
		}
	}
	return out, rows.Err()
}
func sameTagIDs(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[int]bool{}
	for _, id := range a {
		set[id] = true
	}
	for _, id := range b {
		if !set[id] {
			return false
		}
	}
	return true
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
	if !c.TagSync.Active() && len(managed) == 0 {
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
	desired := map[int]map[string]bool{}
	if c.TagSync.Active() {
		desired, err = s.desiredTags(ctx, service, c.TagSync)
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
	for _, tag := range definitions {
		if tag.ID <= 0 || tag.Label == "" {
			return 0, false, errors.New("Invalid tag catalog")
		}
		key := strings.ToLower(tag.Label)
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
		if c.TagSync.Active() {
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
	if err = tx.Commit(); err != nil {
		return 0, false, err
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
	type batch struct{ ids, tags []int }
	batches := map[string]*batch{}
	for _, item := range media {
		if !managed[item.ID] {
			continue
		}
		want := []int{}
		for label := range desired[item.ID] {
			want = append(want, labels[label])
		}
		sort.Ints(want)
		if sameTagIDs(want, item.Tags) {
			continue
		}
		key := fmt.Sprint(want)
		if batches[key] == nil {
			batches[key] = &batch{tags: want}
		}
		batches[key].ids = append(batches[key].ids, item.ID)
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
			if err = client.writeTags(ctx, editor, "PUT", map[string]any{idsKey: b.ids[start:end], "tags": b.tags, "applyTags": "replace", "moveFiles": false}, &applied); err != nil {
				return changed, false, err
			}
			expected := map[int]bool{}
			for _, id := range b.ids[start:end] {
				expected[id] = true
			}
			for _, item := range applied {
				if !expected[item.ID] || !sameTagIDs(item.Tags, b.tags) {
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
