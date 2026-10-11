package app

import (
	"net/url"
	"reflect"
	"testing"
)

func TestOwnedSyncSnapshotKeepsTracksAndReleasesInputs(t *testing.T) {
	s := testStore(t)
	f := sample(1, "A.2160p.x265.DV.HDR10.mkv")
	applyEmby(&f, []embyStream{{Type: "Audio", Index: 1, Codec: "dts", Language: "ger"}, {Type: "Audio", Index: 2, Codec: "ac3", Language: "eng"}})
	files := []File{f}
	// Other store callers retain their input and can reuse it.
	if err := s.Replace("radarr", files, SyncStatus{Count: 1}); err != nil {
		t.Fatal(err)
	}
	if files[0].ID != 1 || len(files[0].AudioTracks) != 2 {
		t.Fatal("regular Replace consumed its caller's input")
	}
	if err := s.replaceSyncSnapshot("radarr", files, SyncStatus{Count: 1}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files[0], File{}) {
		t.Fatal("owned snapshot was retained")
	}
	result, err := s.Query(url.Values{"service": {"radarr"}, "audio": {"DTS", "AC-3"}, "audioOp": {"all"}}, "matches", false)
	if err != nil || result.Total != 1 || len(result.Files[0].AudioTracks) != 2 || result.Files[0].EmbyMatch != "matched" {
		t.Fatal("stored track data changed", result, err)
	}
}

func TestOwnedSyncSnapshotFailureRollsBackAndReleasesInputs(t *testing.T) {
	s := testStore(t)
	if err := s.Replace("radarr", []File{sample(9, "Existing.1080p.mkv")}, SyncStatus{Count: 1}); err != nil {
		t.Fatal(err)
	}
	f := sample(1, "New.2160p.mkv")
	files := []File{f, f}
	if s.replaceSyncSnapshot("radarr", files, SyncStatus{Count: 2}) == nil {
		t.Fatal("duplicate file accepted")
	}
	for _, f := range files {
		if !reflect.DeepEqual(f, File{}) {
			t.Fatal("failed snapshot retained input")
		}
	}
	result, err := s.Query(url.Values{"service": {"radarr"}}, "matches", false)
	if err != nil || result.Total != 1 || result.Files[0].ID != 9 {
		t.Fatal("failed snapshot damaged the previous cache", result, err)
	}
	var status SyncStatus
	if err := s.Get("sync:radarr", &status); err != nil || status.Count != 1 {
		t.Fatal("failed snapshot changed persisted sync status", status, err)
	}
}
