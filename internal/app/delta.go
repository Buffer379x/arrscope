package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

var ErrArrNotFound = errors.New("Service entry was not found")

type DeltaScope struct {
	MediaID int
	FileIDs []int
	All     bool
}

func deleteScopedFiles(tx *sql.Tx, service string, scope *DeltaScope) (sql.Result, error) {
	if scope == nil {
		return tx.Exec("DELETE FROM files WHERE service=?", service)
	}
	if scope.MediaID <= 0 {
		return nil, errors.New("Invalid delta scope")
	}
	if scope.All {
		return tx.Exec("DELETE FROM files WHERE service=? AND media_id=?", service, scope.MediaID)
	}
	args := []any{service, scope.MediaID}
	marks := []string{}
	for _, id := range scope.FileIDs {
		if id <= 0 {
			return nil, errors.New("Invalid file ID")
		}
		marks = append(marks, "?")
		args = append(args, id)
	}
	if len(marks) == 0 {
		return nil, errors.New("Empty delta scope")
	}
	return tx.Exec("DELETE FROM files WHERE service=? AND media_id=? AND id IN ("+strings.Join(marks, ",")+")", args...)
}
func (c *ArrClient) FetchDelta(ctx context.Context, service string, scope *DeltaScope) ([]File, error) {
	if service == "radarr" {
		var m struct {
			ID        int    `json:"id"`
			Title     string `json:"title"`
			Year      int    `json:"year"`
			TitleSlug string `json:"titleSlug"`
			TMDBID    int    `json:"tmdbId"`
		}
		e := c.get(ctx, "movie/"+strconv.Itoa(scope.MediaID), nil, &m)
		if errors.Is(e, ErrArrNotFound) {
			scope.All = true
			return []File{}, nil
		}
		if e != nil {
			return nil, e
		}
		if m.ID != scope.MediaID || m.Title == "" {
			return nil, errors.New("Invalid movie response")
		}
		scope.All = true
		var files []arrFile
		if e = c.get(ctx, "moviefile", url.Values{"movieId": {strconv.Itoa(m.ID)}}, &files); e != nil {
			return nil, e
		}
		out := []File{}
		seen := map[int]bool{}
		for _, a := range files {
			if a.ID <= 0 || seen[a.ID] || a.MovieID != m.ID || basename(a) == "" || a.Size < 0 {
				return nil, errors.New("Invalid movie file response")
			}
			seen[a.ID] = true
			f := File{ID: a.ID, Service: service, MediaID: m.ID, Title: m.Title, TitleSlug: movieSlug(m.TitleSlug, m.TMDBID), Years: []int{}, Episodes: []Episode{}, Filename: basename(a), SourcePath: a.Path, Size: a.Size}
			if m.Year > 0 {
				f.Years = []int{m.Year}
			}
			a.MediaInfo.Quality = a.Quality.Quality.Name
			Enrich(&f, a.MediaInfo)
			out = append(out, f)
		}
		return out, nil
	}
	var series struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		TitleSlug string `json:"titleSlug"`
	}
	e := c.get(ctx, "series/"+strconv.Itoa(scope.MediaID), nil, &series)
	if errors.Is(e, ErrArrNotFound) {
		scope.All = true
		return []File{}, nil
	}
	if e != nil {
		return nil, e
	}
	if series.ID != scope.MediaID || series.Title == "" {
		return nil, errors.New("Invalid series response")
	}
	q := url.Values{}
	if scope.All {
		q.Set("seriesId", strconv.Itoa(scope.MediaID))
	} else {
		for _, id := range scope.FileIDs {
			q.Add("episodeFileIds", strconv.Itoa(id))
		}
	}
	var files []arrFile
	if e = c.get(ctx, "episodefile", q, &files); e != nil {
		return nil, e
	}
	allowed := map[int]bool{}
	for _, id := range scope.FileIDs {
		allowed[id] = true
	}
	seen := map[int]bool{}
	out := []File{}
	// Two requests at most are in flight. Episode mappings are limited to one file.
	for _, a := range files {
		if a.ID <= 0 || seen[a.ID] || (!scope.All && !allowed[a.ID]) || (a.SeriesID != 0 && a.SeriesID != scope.MediaID) || basename(a) == "" || a.Size < 0 {
			return nil, errors.New("Invalid episode file response")
		}
		seen[a.ID] = true
		var episodes []struct {
			ID       int    `json:"id"`
			SeriesID int    `json:"seriesId"`
			FileID   int    `json:"episodeFileId"`
			Number   int    `json:"episodeNumber"`
			Title    string `json:"title"`
			AirDate  string `json:"airDate"`
		}
		if e = c.get(ctx, "episode", url.Values{"episodeFileId": {strconv.Itoa(a.ID)}}, &episodes); e != nil {
			return nil, e
		}
		eps := []Episode{}
		years := map[int]bool{}
		titles := []string{}
		for _, ep := range episodes {
			if ep.ID <= 0 || ep.FileID != a.ID || (ep.SeriesID != 0 && ep.SeriesID != scope.MediaID) {
				return nil, errors.New("Invalid episode mapping")
			}
			year := 0
			if len(ep.AirDate) >= 4 {
				year, _ = strconv.Atoi(ep.AirDate[:4])
			}
			if year > 0 {
				years[year] = true
			}
			eps = append(eps, Episode{ep.ID, ep.Number, ep.Title, year})
		}
		sort.Slice(eps, func(i, j int) bool { return eps[i].Number < eps[j].Number })
		for _, ep := range eps {
			titles = append(titles, fmt.Sprintf("E%02d · %s", ep.Number, ep.Title))
		}
		ys := []int{}
		for y := range years {
			ys = append(ys, y)
		}
		sort.Ints(ys)
		title := strings.Join(titles, " / ")
		if title == "" {
			title = basename(a)
		}
		f := File{ID: a.ID, Service: service, MediaID: series.ID, Title: title, TitleSlug: safeTitleSlug(series.TitleSlug), Series: series.Title, Season: a.Season, Episodes: eps, Years: ys, Filename: basename(a), SourcePath: a.Path, Size: a.Size}
		a.MediaInfo.Quality = a.Quality.Quality.Name
		Enrich(&f, a.MediaInfo)
		out = append(out, f)
	}
	return out, nil
}
func (s *Store) preserveDeltaEmby(ctx context.Context, service string, files []File, enabled bool) error {
	for i := range files {
		f := &files[i]
		var raw string
		err := s.DB.QueryRowContext(ctx, "SELECT data FROM files WHERE service=? AND id=?", service, f.ID).Scan(&raw)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var old File
		if err == nil {
			if e := json.Unmarshal([]byte(raw), &old); e != nil {
				return e
			}
		}
		if enabled && old.ID == f.ID && old.Filename == f.Filename && old.Size == f.Size {
			f.EmbyMatch = old.EmbyMatch
			f.MediaServers = old.MediaServers
			f.MediaOrigin = old.MediaOrigin
			f.AudioTracks = old.AudioTracks
			e := f.Evidence["hdr"]
			e.Emby = old.Evidence["hdr"].Emby
			f.Evidence["hdr"] = e
			f.RecognitionRevision = ""
		} else if enabled {
			f.EmbyMatch = "pending"
		} else {
			f.EmbyMatch = "unconfigured"
		}
	}
	return nil
}
