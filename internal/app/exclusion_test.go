package app

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"
)

func TestExclusionExampleAndSeasonContext(t *testing.T) {
	store := testStore(t)
	names := []string{"A.1080p.x265.BluRay.mkv", "B.1080p.x265.WEB-DL.mkv", "C.1080p.x265.WEBRip.mkv", "D.1080p.x265.HDTV.mkv", "E.1080p.x265.mkv", "F.2160p.x265.WEBRip.mkv", "G.1080p.x264.WEBRip.mkv"}
	files := make([]File, len(names))
	for i, name := range names {
		files[i] = sample(i+1, name)
		files[i].Series = "Show"
		files[i].Season = 1
	}
	for _, service := range []string{"radarr", "sonarr"} {
		if err := store.Replace(service, files, SyncStatus{}); err != nil {
			t.Fatal(err)
		}
		q := url.Values{"service": {service}, "resolution": {"1080p"}, "codec": {"x265"}, "source": {"BluRay", "WEB-DL"}, "sourceOp": {"not"}}
		assertIDs := func(mode string, all bool, want []int) Results {
			t.Helper()
			result, err := store.Query(q, mode, all)
			if err != nil {
				t.Fatal(err)
			}
			ids := []int{}
			for _, f := range result.Files {
				if f.Matched {
					ids = append(ids, f.ID)
				}
			}
			if !reflect.DeepEqual(ids, want) || result.Total != len(want) {
				t.Fatalf("%s %s: got %v total %d want %v", service, mode, ids, result.Total, want)
			}
			return result
		}
		assertIDs("matches", false, []int{3, 4, 5})
		assertIDs("matches", true, []int{3, 4, 5})
		if service == "sonarr" {
			context := assertIDs("season", false, []int{3, 4, 5})
			if context.DisplayTotal != 7 || len(context.Files) != 7 {
				t.Fatal("missing season context", context)
			}
			export := assertIDs("season", true, []int{3, 4, 5})
			if len(export.Files) != 3 {
				t.Fatal("export includes context")
			}
		}
		q.Add("source", "__unknown__")
		assertIDs("matches", false, []int{3, 4})
		q.Set("codecOp", "not")
		assertIDs("matches", false, []int{7})
		q.Del("source")
		q.Del("codecOp")
		assertIDs("matches", false, []int{1, 2, 3, 4, 5}) // No values means no exclusion.
	}
}

func TestExclusionEveryTraitAndCompoundValues(t *testing.T) {
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			store := testStore(t)
			known, other, unknown := sample(1, "A.mkv"), sample(2, "B.mkv"), sample(3, "C.mkv")
			set := func(f *File, value string) {
				reflect.ValueOf(f).Elem().FieldByName(map[string]string{"resolution": "Resolution", "codec": "Codec", "family": "Family", "source": "Source", "processing": "Processing", "hdr": "HDR", "bitDepth": "BitDepth", "audio": "Audio", "channels": "Channels", "language": "Language"}[field]).SetString(value)
			}
			set(&known, "Chosen")
			set(&other, "Other")
			set(&unknown, "")
			value := "Chosen"
			if field == "language" {
				set(&known, "Deutsch, Englisch")
				value = "Deutsch"
			}
			if field == "hdr" {
				set(&known, "Dolby Vision + HDR10+")
				value = "Dolby Vision"
			}
			if err := store.Replace("radarr", []File{known, other, unknown}, SyncStatus{}); err != nil {
				t.Fatal(err)
			}
			cases := []struct {
				values []string
				op     string
				want   []int
			}{
				{[]string{value}, "not", []int{2, 3}},
				{[]string{value, "__unknown__"}, "not", []int{2}},
				{[]string{value, "Other"}, "not", []int{3}},
				{[]string{"__unknown__"}, "not", []int{1, 2}},
				{[]string{value, "Other"}, "eq", []int{1, 2}},
				{[]string{value}, "", []int{1}},
				{[]string{"x') OR 1=1 --"}, "not", []int{1, 2, 3}},
			}
			if field == "language" {
				cases = append(cases, struct {
					values []string
					op     string
					want   []int
				}{[]string{"Deut"}, "not", []int{1, 2, 3}})
			}
			if field == "hdr" {
				cases = append(cases, struct {
					values []string
					op     string
					want   []int
				}{[]string{"HDR10"}, "not", []int{1, 2, 3}})
			}
			for _, tc := range cases {
				q := url.Values{"service": {"radarr"}, field: tc.values, field + "Op": {tc.op}}
				result, err := store.Query(q, "matches", false)
				if err != nil {
					t.Fatal(err)
				}
				ids := []int{}
				for _, f := range result.Files {
					ids = append(ids, f.ID)
				}
				if !reflect.DeepEqual(ids, tc.want) {
					t.Fatalf("%v => %v, want %v", q, ids, tc.want)
				}
			}
			for _, op := range []string{"NOT", "neq", "not); DROP TABLE files;--"} {
				if _, _, err := where(url.Values{"service": {"radarr"}, field + "Op": {op}}); err == nil {
					t.Fatal("invalid operator accepted", op)
				}
			}
			if _, _, err := where(url.Values{"service": {"radarr"}, field + "Op": {"eq", "not"}}); err == nil {
				t.Fatal("ambiguous operators accepted")
			}
		})
	}
}

func TestExclusionPagination(t *testing.T) {
	store := testStore(t)
	files := make([]File, 220)
	for i := range files {
		files[i] = sample(i+1, fmt.Sprintf("%03d.1080p.x265.WEBRip.mkv", i))
		if i%2 == 0 {
			files[i].Source = "BluRay"
		}
	}
	if err := store.Replace("radarr", files, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"service": {"radarr"}, "source": {"BluRay"}, "sourceOp": {"not"}, "page": {"2"}}
	page, err := store.Query(q, "matches", false)
	if err != nil || page.Total != 110 || len(page.Files) != 10 {
		t.Fatal("negative pagination", page, err)
	}
	all, err := store.Query(q, "matches", true)
	if err != nil || len(all.Files) != 110 {
		t.Fatal("all negative matches", err)
	}
}
