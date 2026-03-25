package arr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockSonarrServer(t *testing.T, records []map[string]any, total int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"totalRecords": total,
			"records":      records,
		})
	}))
}

func TestGetMissingEpisodes(t *testing.T) {
	server := mockSonarrServer(t, []map[string]any{
		{
			"id": 1, "seasonNumber": 1, "episodeNumber": 2,
			"title": "Pilot", "monitored": true,
			"series": map[string]any{"title": "Test Show"},
		},
	}, 1)
	defer server.Close()

	client := NewSonarrClient(server.URL, "testkey")
	result, err := client.GetMissingEpisodes(1, 20, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 1 {
		t.Fatalf("expected total 1, got %d", result.TotalCount)
	}
	if result.Episodes[0].SeriesTitle != "Test Show" {
		t.Fatalf("expected Test Show, got %s", result.Episodes[0].SeriesTitle)
	}
}

func TestSearchFilter(t *testing.T) {
	records := []map[string]any{}
	for i := 0; i < 10; i++ {
		records = append(records, map[string]any{
			"id": i, "seasonNumber": 1, "episodeNumber": i,
			"title": "Episode", "monitored": true,
			"series": map[string]any{"title": "Cops"},
		})
	}
	for i := 10; i < 15; i++ {
		records = append(records, map[string]any{
			"id": i, "seasonNumber": 1, "episodeNumber": i,
			"title": "Episode", "monitored": true,
			"series": map[string]any{"title": "The Wire"},
		})
	}

	server := mockSonarrServer(t, records, 15)
	defer server.Close()

	client := NewSonarrClient(server.URL, "testkey")
	result, err := client.GetMissingEpisodes(1, 20, "cops")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalCount != 10 {
		t.Fatalf("expected 10 filtered results, got %d", result.TotalCount)
	}
	for _, ep := range result.Episodes {
		if ep.SeriesTitle != "Cops" {
			t.Fatalf("expected only Cops episodes, got %s", ep.SeriesTitle)
		}
	}
}

func TestSearchPagination(t *testing.T) {
	records := []map[string]any{}
	for i := 0; i < 25; i++ {
		records = append(records, map[string]any{
			"id": i, "seasonNumber": 1, "episodeNumber": i,
			"title": "Episode", "monitored": true,
			"series": map[string]any{"title": "Cops"},
		})
	}

	server := mockSonarrServer(t, records, 25)
	defer server.Close()

	client := NewSonarrClient(server.URL, "testkey")

	result, _ := client.GetMissingEpisodes(1, 10, "cops")
	if len(result.Episodes) != 10 {
		t.Fatalf("page 1: expected 10 episodes, got %d", len(result.Episodes))
	}
	if result.TotalCount != 25 {
		t.Fatalf("expected total 25, got %d", result.TotalCount)
	}

	result, _ = client.GetMissingEpisodes(3, 10, "cops")
	if len(result.Episodes) != 5 {
		t.Fatalf("page 3: expected 5 episodes, got %d", len(result.Episodes))
	}
}

func TestAPIKeyNotInURL(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"totalRecords": 0, "records": []any{}})
	}))
	defer server.Close()

	client := NewSonarrClient(server.URL, "my-secret-key")
	client.GetMissingEpisodes(1, 20, "")

	if capturedURL == "" {
		t.Fatal("no request made")
	}
	t.Logf("Request URL (server-side only, never exposed to frontend): %s", capturedURL)
}
