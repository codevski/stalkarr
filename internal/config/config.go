package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type SonarrInstance struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

type AuthConfig struct {
	Username         string `json:"username"`
	PasswordHash     string `json:"password_hash"`
	RefreshTokenHash string `json:"refresh_token_hash,omitempty"`
}

type Config struct {
	Auth   AuthConfig       `json:"auth"`
	Sonarr []SonarrInstance `json:"sonarr"`
	Hunt   HuntConfig       `json:"hunt"`
}

type HuntConfig struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"intervalMinutes"`
	EpisodesPerRun  int  `json:"episodesPerRun"`
	CooldownHours   int  `json:"cooldownHours"`
}

var (
	mu       sync.RWMutex
	current  Config
	dataPath string
)

func Init(dir string) error {
	dataPath = filepath.Join(dir, "config.json")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return Save(Config{})
	}
	return load()
}

func load() error {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.Open(dataPath)
	if err != nil {
		return err
	}
	defer f.Close()

	current = Config{
		Hunt: HuntConfig{
			Enabled:         false,
			IntervalMinutes: 60,
			EpisodesPerRun:  10,
			CooldownHours:   24,
		},
	}
	return json.NewDecoder(f).Decode(&current)
}

func Save(c Config) error {
	mu.Lock()
	defer mu.Unlock()
	current = c
	f, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

func Get() Config {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func GetSonarrInstance(id string) (SonarrInstance, bool) {
	mu.RLock()
	defer mu.RUnlock()
	for _, s := range current.Sonarr {
		if s.ID == id {
			return s, true
		}
	}
	return SonarrInstance{}, false
}
