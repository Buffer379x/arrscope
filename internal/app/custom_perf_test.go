package app

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestPerformance50KCustom(t *testing.T) {
	if os.Getenv("SCOPEARR_PERF") != "1" {
		t.Skip("set SCOPEARR_PERF=1")
	}
	s := testStore(t)
	ctx := context.Background()
	files := make([]File, 50000)
	for i := range files {
		files[i] = File{ID: i + 1, MediaID: i + 1, Title: fmt.Sprintf("Movie %06d", i), Filename: fmt.Sprintf("Movie.%06d.1080p.x265.mkv", i), Resolution: "1080p", Codec: "x265", Years: []int{2020}}
	}
	if err := s.Replace("radarr", files, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	a, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Archive"})
	b, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Favorites"})
	start := time.Now()
	n, err := s.ApplyCustom(ctx, url.Values{"service": {"radarr"}}, map[string]bool{strconv.FormatInt(a.ID, 10): true, strconv.FormatInt(b.ID, 10): true})
	if err != nil || n != 50000 {
		t.Fatal(n, err)
	}
	t.Logf("Custom tags bulk assignment: %s", time.Since(start))
	times := []time.Duration{}
	for i := 0; i < 30; i++ {
		q := url.Values{"service": {"radarr"}, "custom": {strconv.FormatInt(a.ID, 10), strconv.FormatInt(b.ID, 10)}, "customOp": {"all"}, "resolution": {"1080p"}, "codec": {"x265"}}
		start = time.Now()
		out, err := s.Query(q, "matches", false)
		if err != nil || out.Total != 50000 || len(out.Files) != 100 {
			t.Fatal(out.Total, err)
		}
		times = append(times, time.Since(start))
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	t.Logf("Custom tags 50,000-file filter p95: %s", times[28])
	if times[28] > 200*time.Millisecond {
		t.Fatal("custom filter p95 above 200 ms", times[28])
	}
}
