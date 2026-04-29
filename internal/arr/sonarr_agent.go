package arr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RunResult struct {
	CommandID int    `json:"commandId"`
	Message   string `json:"message"`
}

func (c *SonarrClient) TriggerEpisodeSearch(episodeIDs []int) (RunResult, error) {
	if len(episodeIDs) == 0 {
		return RunResult{}, fmt.Errorf("no episode IDs provided")
	}

	payload := map[string]any{
		"name":       "EpisodeSearch",
		"episodeIds": episodeIDs,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return RunResult{}, fmt.Errorf("marshal command: %w", err)
	}

	url := fmt.Sprintf("%s/api/v3/command?apikey=%s", c.BaseURL, c.APIKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return RunResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return RunResult{}, fmt.Errorf("sonarr unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return RunResult{}, fmt.Errorf("sonarr returned %d: %s", resp.StatusCode, string(b))
	}

	var raw struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return RunResult{Message: fmt.Sprintf("Searching for %d episode(s)", len(episodeIDs))}, nil
	}

	return RunResult{
		CommandID: raw.ID,
		Message:   fmt.Sprintf("Searching for %d episode(s)", len(episodeIDs)),
	}, nil
}
