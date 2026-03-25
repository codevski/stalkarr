package version

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

var Version = "dev"

const repoOwner = "codevski"
const repoName = "stalkarr"

type updateCache struct {
	latest    string
	fetchedAt time.Time
	mu        sync.Mutex
}

var cache updateCache

func CheckForUpdate() (latest string, hasUpdate bool, err error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if cache.latest != "" && time.Since(cache.fetchedAt) < 6*time.Hour {
		return cache.latest, isNewer(cache.latest, Version), nil
	}

	url := "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false, err
	}

	cache.latest = release.TagName
	cache.fetchedAt = time.Now()

	return cache.latest, isNewer(cache.latest, Version), nil
}

func isNewer(latest, current string) bool {
	if current == "dev" {
		return false
	}
	return latest != "" && latest != current
}
