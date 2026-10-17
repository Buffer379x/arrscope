package app

import (
	"context"
	"database/sql"
	"strings"
)

// Keep historical display values when their source data changes. Reading the
// catalog is then a read-only operation, including on a cold cache.
func rememberDisplayTags(ctx context.Context, tx *sql.Tx, rules []RecognitionRule) error {
	active, err := displayTags(ctx, tx, rules)
	if err != nil {
		return err
	}
	for field, values := range active {
		if !catalogField(field) {
			continue
		}
		if len(values) == 0 {
			continue
		}
		for _, value := range values {
			if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO display_tag_catalog(field,value) VALUES(?,?)", field, value); err != nil {
				return err
			}
		}
		// Only tombstones need a presence check. Empty catalogs avoid rescanning
		// the file cache, and a newly assigned tag restores its display entry.
		present := "f." + field + "=c.value"
		if field == "hdr" || field == "audio" || field == "channels" {
			present = "instr(' + '||f." + field + "||' + ',' + '||c.value||' + ')>0"
		}
		args := []any{field}
		for _, value := range values {
			args = append(args, value)
		}
		marks := strings.TrimSuffix(strings.Repeat("?,", len(values)), ",")
		query := "UPDATE display_tag_catalog AS c SET deleted=0 WHERE c.field=? AND c.deleted=1 AND c.value IN (" + marks + ") AND EXISTS(SELECT 1 FROM files f WHERE (c.value<>'__unknown__' AND (" + present + ")) OR (c.value='__unknown__' AND (f." + field + "='' OR CASE WHEN json_valid(f.data) THEN EXISTS(SELECT 1 FROM json_each(f.data,'$.unknownTraits') WHERE value=c.field) ELSE 0 END)))"
		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}
