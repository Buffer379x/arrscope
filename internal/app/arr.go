package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ArrClient struct {
	Base string
	Key  string
	HTTP *http.Client
}

func NewArr(c ServiceConfig) *ArrClient {
	return &ArrClient{c.URL, c.APIKey, &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return errors.New("Weiterleitung nicht erlaubt; direkte Dienstadresse verwenden")
	}}}
}
func ValidateConfig(c ServiceConfig) error {
	if _, err := parsePathMappings(c.PathMappings); err != nil {
		return err
	}
	validInterval := map[int]bool{0: true, 15: true, 30: true, 60: true, 360: true, 1440: true}
	if !validInterval[c.Interval] {
		return errors.New("Ungültiges Sync-Intervall")
	}
	if !c.Enabled && c.URL == "" {
		return nil
	}
	u, e := url.Parse(c.URL)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("Dienstadresse muss eine HTTP(S)-URL ohne Zugangsdaten, Query oder Fragment sein")
	}
	if c.Enabled && c.APIKey == "" {
		return errors.New("API-Key fehlt")
	}
	return nil
}
func (c *ArrClient) get(ctx context.Context, path string, q url.Values, out any) error {
	allowed := map[string]bool{"system/status": true, "movie": true, "moviefile": true, "series": true, "episode": true, "episodefile": true}
	if !allowed[path] {
		return errors.New("API-Endpunkt nicht erlaubt")
	}
	u, e := url.Parse(strings.TrimRight(c.Base, "/") + "/api/v3/" + path)
	if e != nil {
		return errors.New("Ungültige Dienstadresse")
	}
	u.RawQuery = q.Encode()
	return c.readJSON(ctx, u, "X-Api-Key", out)
}

// All integrations share GET-only requests, timeout, bounded retries and header authentication.
func (c *ArrClient) readJSON(ctx context.Context, u *url.URL, keyHeader string, out any) error {
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * 250 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		r, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if e != nil {
			return errors.New("Anfrage konnte nicht erstellt werden")
		}
		r.Header.Set(keyHeader, c.Key)
		r.Header.Set("Accept", "application/json")
		res, e := c.HTTP.Do(r)
		if e != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt < 2 {
				continue
			}
			return errors.New("Dienst nicht erreichbar, Zeitlimit oder TLS-Prüfung fehlgeschlagen")
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			if res.StatusCode >= 500 && attempt < 2 {
				continue
			}
			if res.StatusCode == 401 || res.StatusCode == 403 {
				return errors.New("API-Key wurde abgelehnt")
			}
			return fmt.Errorf("Dienst meldet HTTP %d", res.StatusCode)
		}
		e = decodeServiceJSON(res.Body, out, 64<<20)
		res.Body.Close()
		if e != nil {
			return errors.New("Ungültige oder zu große API-Antwort")
		}
		return nil
	}
	return errors.New("API-Anfrage fehlgeschlagen")
}

// Require a complete non-null object/list without holding a second response copy.
func decodeServiceJSON(body io.Reader, out any, max int64) error {
	limited := &io.LimitedReader{R: body, N: max + 1}
	reader := bufio.NewReader(limited)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		if b != '[' && b != '{' {
			return errors.New("API-Antwort muss eine Liste oder ein Objekt sein")
		}
		if err = reader.UnreadByte(); err != nil {
			return err
		}
		break
	}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("Unvollständige oder mehrteilige API-Antwort")
	}
	if limited.N == 0 {
		return errors.New("API-Antwort überschreitet das Größenlimit")
	}
	return nil
}

func readPair(ctx context.Context, first, second func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, read := range []func(context.Context) error{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := read(ctx)
			if err != nil {
				cancel()
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	var result error
	for err := range errs {
		if err != nil && (result == nil || errors.Is(result, context.Canceled)) {
			result = err
		}
	}
	return result
}
func (c *ArrClient) Test(ctx context.Context) (string, error) {
	var out struct {
		Version string `json:"version"`
	}
	e := c.get(ctx, "system/status", nil, &out)
	return out.Version, e
}

type quality struct {
	Quality struct {
		Name string `json:"name"`
	} `json:"quality"`
}
type arrFile struct {
	ID           int     `json:"id"`
	MovieID      int     `json:"movieId"`
	SeriesID     int     `json:"seriesId"`
	Season       int     `json:"seasonNumber"`
	RelativePath string  `json:"relativePath"`
	Path         string  `json:"path"`
	Size         int64   `json:"size"`
	Quality      quality `json:"quality"`
	MediaInfo    APIInfo `json:"mediaInfo"`
}

func basename(f arrFile) string {
	p := f.RelativePath
	if p == "" {
		p = f.Path
	}
	p = strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(p, "/")
	return parts[len(parts)-1]
}
func (c *ArrClient) Fetch(ctx context.Context, service string) ([]File, error) {
	if service == "radarr" {
		return c.movies(ctx)
	}
	if service == "sonarr" {
		return c.series(ctx)
	}
	return nil, errors.New("Ungültiger Dienst")
}
func (c *ArrClient) movies(ctx context.Context) ([]File, error) {
	var movies []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Year  int    `json:"year"`
	}
	if e := c.get(ctx, "movie", url.Values{"excludeLocalCovers": {"true"}}, &movies); e != nil {
		return nil, e
	}
	index := map[int]struct {
		title string
		year  int
	}{}
	for _, m := range movies {
		if m.ID <= 0 || m.Title == "" {
			return nil, errors.New("Unvollständiger Filmkatalog")
		}
		if _, duplicate := index[m.ID]; duplicate {
			return nil, errors.New("Doppelte Film-ID im Katalog")
		}
		index[m.ID] = struct {
			title string
			year  int
		}{m.Title, m.Year}
	}
	out := []File{}
	seenFiles := map[int]bool{}
	for start := 0; start < len(movies); start += 100 {
		end := start + 100
		if end > len(movies) {
			end = len(movies)
		}
		q := url.Values{}
		batch := map[int]bool{}
		for _, m := range movies[start:end] {
			batch[m.ID] = true
			q.Add("movieId", strconv.Itoa(m.ID))
		}
		var files []arrFile
		if e := c.get(ctx, "moviefile", q, &files); e != nil {
			return nil, e
		}
		for _, a := range files {
			if a.ID <= 0 || seenFiles[a.ID] || !batch[a.MovieID] || basename(a) == "" || a.Size < 0 {
				return nil, errors.New("Ungültige oder doppelte Filmdatei")
			}
			seenFiles[a.ID] = true
			m, ok := index[a.MovieID]
			if !ok {
				return nil, errors.New("Filmzuordnung während Sync geändert; bitte erneut synchronisieren")
			}
			f := File{ID: a.ID, Service: "radarr", MediaID: a.MovieID, Title: m.title, Filename: basename(a), SourcePath: a.Path, Size: a.Size, Years: []int{}, Episodes: []Episode{}}
			if m.year > 0 {
				f.Years = []int{m.year}
			}
			a.MediaInfo.Quality = a.Quality.Quality.Name
			Enrich(&f, a.MediaInfo)
			out = append(out, f)
		}
	}
	return out, nil
}
func (c *ArrClient) series(ctx context.Context) ([]File, error) {
	var series []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if e := c.get(ctx, "series", nil, &series); e != nil {
		return nil, e
	}
	out := []File{}
	seenSeries, seenFiles := map[int]bool{}, map[int]bool{}
	for _, s := range series {
		if s.ID <= 0 || s.Title == "" || seenSeries[s.ID] {
			return nil, errors.New("Ungültiger Serienkatalog")
		}
		seenSeries[s.ID] = true
		q := url.Values{"seriesId": {strconv.Itoa(s.ID)}}
		var files []arrFile
		var episodes []struct {
			ID      int    `json:"id"`
			FileID  int    `json:"episodeFileId"`
			Number  int    `json:"episodeNumber"`
			Title   string `json:"title"`
			AirDate string `json:"airDate"`
		}
		if e := readPair(ctx, func(ctx context.Context) error { return c.get(ctx, "episodefile", q, &files) }, func(ctx context.Context) error { return c.get(ctx, "episode", q, &episodes) }); e != nil {
			return nil, e
		}
		byFile := map[int][]Episode{}
		for _, ep := range episodes {
			year := 0
			if len(ep.AirDate) >= 4 {
				year, _ = strconv.Atoi(ep.AirDate[:4])
			}
			byFile[ep.FileID] = append(byFile[ep.FileID], Episode{ep.ID, ep.Number, ep.Title, year})
		}
		for _, a := range files {
			if a.ID <= 0 || seenFiles[a.ID] || (a.SeriesID != 0 && a.SeriesID != s.ID) || basename(a) == "" || a.Size < 0 {
				return nil, errors.New("Ungültige oder doppelte Episodendatei")
			}
			seenFiles[a.ID] = true
			eps := byFile[a.ID]
			sort.Slice(eps, func(i, j int) bool { return eps[i].Number < eps[j].Number })
			ys := map[int]bool{}
			var titles []string
			for _, ep := range eps {
				if ep.Year > 0 {
					ys[ep.Year] = true
				}
				titles = append(titles, fmt.Sprintf("E%02d · %s", ep.Number, ep.Title))
			}
			years := []int{}
			for y := range ys {
				years = append(years, y)
			}
			sort.Ints(years)
			title := strings.Join(titles, " / ")
			if title == "" {
				title = basename(a)
			}
			if eps == nil {
				eps = []Episode{}
			}
			f := File{ID: a.ID, Service: "sonarr", MediaID: s.ID, Title: title, Series: s.Title, Season: a.Season, Episodes: eps, Years: years, Filename: basename(a), SourcePath: a.Path, Size: a.Size}
			a.MediaInfo.Quality = a.Quality.Quality.Name
			Enrich(&f, a.MediaInfo)
			out = append(out, f)
		}
	}
	return out, nil
}
