package app

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

func addAudioStatistics(ctx context.Context, tx *sql.Tx, out *Statistics) error {
	out.FileTotals = map[string]int{}
	for _, scope := range []string{"all", "radarr", "sonarr"} {
		filter := ""
		args := []any{}
		if scope != "all" {
			filter = " WHERE service=?"
			args = append(args, scope)
		}
		var total int
		if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM files"+filter, args...).Scan(&total); e != nil {
			return e
		}
		out.FileTotals[scope] = total
		counts := map[string]int{}
		rows, e := tx.QueryContext(ctx, "SELECT audio,count(*) FROM files"+filter+" GROUP BY audio", args...)
		if e != nil {
			return e
		}
		for rows.Next() {
			var value string
			var n int
			if e = rows.Scan(&value, &n); e != nil {
				rows.Close()
				return e
			}
			seen := map[string]bool{}
			for _, codec := range strings.Split(value, " + ") {
				if codec != "" && !seen[codec] {
					counts[codec] += n
					seen[codec] = true
				}
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		where := " WHERE (audio='' OR EXISTS(SELECT 1 FROM json_each(files.data,'$.unknownTraits') u WHERE u.value='audio'))"
		if scope != "all" {
			where += " AND service=?"
		}
		var unknown int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files"+where, args...).Scan(&unknown); e != nil {
			return e
		}
		if unknown > 0 {
			counts[""] = unknown
		}
		buckets := []StatisticBucket{}
		for value, n := range counts {
			buckets = append(buckets, StatisticBucket{Value: value, Count: n})
		}
		sort.Slice(buckets, func(i, j int) bool {
			if buckets[i].Count != buckets[j].Count {
				return buckets[i].Count > buckets[j].Count
			}
			return buckets[i].Value < buckets[j].Value
		})
		out.Distributions[scope]["audio"] = buckets
	}
	return nil
}
