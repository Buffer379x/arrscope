package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strconv"
)

// Both custom-tag APIs hold recognitionMu before changing rules and references.
func (s *Store) recognitionWithoutCustomTags(deleted map[int64]bool) (*RecognitionEngine, error) {
	rules := []RecognitionRule{}
	for _, rule := range s.recognition.Load().config.Rules {
		id, _ := strconv.ParseInt(rule.Value, 10, 64)
		if rule.Field != "custom" || !deleted[id] {
			rules = append(rules, rule)
		}
	}
	return CompileRecognition(rules)
}

func cleanupDeletedCustomTags(ctx context.Context, tx *sql.Tx, deleted map[int64]bool, engine *RecognitionEngine) error {
	raw, err := json.Marshal(engine.Config())
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO kv(key,value) VALUES('recognition',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(raw)); err != nil {
		return err
	}
	keys := map[string]bool{}
	for id := range deleted {
		keys[strconv.FormatInt(id, 10)] = true
	}
	var rawSettings string
	err = tx.QueryRowContext(ctx, "SELECT value FROM kv WHERE key='settings'").Scan(&rawSettings)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		var cfg Settings
		if err = json.Unmarshal([]byte(rawSettings), &cfg); err != nil {
			return err
		}
		for id := range keys {
			delete(cfg.TagColors, "custom:"+id)
			delete(cfg.TagTextColors, "custom:"+id)
			delete(cfg.HiddenTags, "custom:"+id)
		}
		raw, err = json.Marshal(cfg)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE kv SET value=? WHERE key='settings'", string(raw)); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,query FROM presets")
	if err != nil {
		return err
	}
	type presetQuery struct {
		id    int64
		query string
	}
	updates := []presetQuery{}
	for rows.Next() {
		var item presetQuery
		if err = rows.Scan(&item.id, &item.query); err != nil {
			rows.Close()
			return err
		}
		q, parseErr := url.ParseQuery(item.query)
		if parseErr != nil {
			continue
		}
		values := q["custom"]
		kept := []string{}
		for _, value := range values {
			if !keys[value] {
				kept = append(kept, value)
			}
		}
		if len(kept) == len(values) {
			continue
		}
		q.Del("custom")
		for _, value := range kept {
			q.Add("custom", value)
		}
		if len(kept) == 0 {
			q.Del("customOp")
		}
		item.query = q.Encode()
		updates = append(updates, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range updates {
		if _, err = tx.ExecContext(ctx, "UPDATE presets SET query=? WHERE id=?", item.query, item.id); err != nil {
			return err
		}
	}
	return nil
}
