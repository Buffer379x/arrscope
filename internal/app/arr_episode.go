package app

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type arrEpisode struct {
	ID       int    `json:"id"`
	SeriesID int    `json:"seriesId"`
	FileID   int    `json:"episodeFileId"`
	Number   int    `json:"episodeNumber"`
	Title    string `json:"title"`
	AirDate  string `json:"airDate"`
}

func episodeMappings(episodes []arrEpisode, seriesID int, fileIDs, seen map[int]bool, requireFile bool) (map[int][]Episode, error) {
	byFile := map[int][]Episode{}
	for _, ep := range episodes {
		if ep.ID <= 0 || seen[ep.ID] || ep.SeriesID != 0 && ep.SeriesID != seriesID || ep.FileID < 0 || (requireFile || ep.FileID != 0) && !fileIDs[ep.FileID] {
			return nil, errors.New("Invalid episode mapping")
		}
		seen[ep.ID] = true
		if ep.FileID == 0 {
			continue
		}
		year := 0
		if len(ep.AirDate) >= 4 {
			year, _ = strconv.Atoi(ep.AirDate[:4])
		}
		byFile[ep.FileID] = append(byFile[ep.FileID], Episode{ep.ID, ep.Number, ep.Title, year})
	}
	return byFile, nil
}

func seriesFile(a arrFile, seriesID int, seriesTitle, slug string, episodes []Episode) File {
	if episodes == nil {
		episodes = []Episode{}
	}
	sort.Slice(episodes, func(i, j int) bool { return episodes[i].Number < episodes[j].Number })
	yearSet := map[int]bool{}
	titles := []string{}
	for _, ep := range episodes {
		if ep.Year > 0 {
			yearSet[ep.Year] = true
		}
		titles = append(titles, fmt.Sprintf("E%02d · %s", ep.Number, ep.Title))
	}
	years := []int{}
	for year := range yearSet {
		years = append(years, year)
	}
	sort.Ints(years)
	title := strings.Join(titles, " / ")
	if title == "" {
		title = basename(a)
	}
	f := File{ID: a.ID, Service: "sonarr", MediaID: seriesID, TitleSlug: safeTitleSlug(slug), Title: title, Series: seriesTitle, Season: a.Season, Episodes: episodes, Years: years, Filename: basename(a), SourcePath: a.Path, Size: a.Size}
	a.MediaInfo.Quality = a.Quality.Quality.Name
	Enrich(&f, a.MediaInfo)
	return f
}
