package arr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type SonarrClient struct {
	BaseURL string
	APIKey  string
}

type Episode struct {
	ID            int    `json:"id"`
	SeriesTitle   string `json:"seriesTitle"`
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Title         string `json:"title"`
	Monitored     bool   `json:"monitored"`
}

type MissingResult struct {
	Episodes   []Episode `json:"episodes"`
	TotalCount int       `json:"totalCount"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
}

func NewSonarrClient(baseURL, apiKey string) *SonarrClient {
	return &SonarrClient{BaseURL: baseURL, APIKey: apiKey}
}

func (c *SonarrClient) GetMissingEpisodes(page, pageSize int, search string) (MissingResult, error) {
	url := fmt.Sprintf(
		"%s/api/v3/wanted/missing?apikey=%s&page=%d&pageSize=%d&includeSeries=true",
		c.BaseURL, c.APIKey, page, pageSize,
	)

	// TODO: Fetch all and filter needs to be optimised!
	if search != "" {
		url = fmt.Sprintf(
			"%s/api/v3/wanted/missing?apikey=%s&page=1&pageSize=500&includeSeries=true",
			c.BaseURL, c.APIKey,
		)
	}

	resp, err := http.Get(url)
	if err != nil {
		return MissingResult{}, err
	}
	defer resp.Body.Close()

	var raw struct {
		TotalRecords int `json:"totalRecords"`
		Records      []struct {
			ID            int    `json:"id"`
			SeasonNumber  int    `json:"seasonNumber"`
			EpisodeNumber int    `json:"episodeNumber"`
			Title         string `json:"title"`
			Monitored     bool   `json:"monitored"`
			Series        struct {
				Title string `json:"title"`
			} `json:"series"`
		} `json:"records"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return MissingResult{}, err
	}

	episodes := make([]Episode, len(raw.Records))
	for i, r := range raw.Records {
		episodes[i] = Episode{
			ID:            r.ID,
			SeriesTitle:   r.Series.Title,
			SeasonNumber:  r.SeasonNumber,
			EpisodeNumber: r.EpisodeNumber,
			Title:         r.Title,
			Monitored:     r.Monitored,
		}
	}

	if search != "" {
		q := strings.ToLower(search)
		filtered := episodes[:0]
		for _, ep := range episodes {
			if strings.Contains(strings.ToLower(ep.SeriesTitle), q) ||
				strings.Contains(strings.ToLower(ep.Title), q) {
				filtered = append(filtered, ep)
			}
		}
		total := len(filtered)
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		return MissingResult{
			Episodes:   filtered[start:end],
			TotalCount: total,
			Page:       page,
			PageSize:   pageSize,
		}, nil
	}

	return MissingResult{
		Episodes:   episodes,
		TotalCount: raw.TotalRecords,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}
