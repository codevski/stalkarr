package config

import (
	"os"
	"testing"
)

func TestInitCreatesConfigFile(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if _, err := os.Stat(dir + "/config.json"); os.IsNotExist(err) {
		t.Fatal("expected config.json to be created")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	Init(dir)

	cfg := Config{
		Sonarr: []SonarrInstance{
			{ID: "sonarr-1", Name: "Sonarr", URL: "http://localhost:8989", APIKey: "testkey"},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := Init(dir); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	loaded := Get()
	if len(loaded.Sonarr) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(loaded.Sonarr))
	}
	if loaded.Sonarr[0].APIKey != "testkey" {
		t.Fatalf("expected apikey testkey, got %s", loaded.Sonarr[0].APIKey)
	}
}

func TestGetSonarrInstance(t *testing.T) {
	dir := t.TempDir()
	Init(dir)

	Save(Config{
		Sonarr: []SonarrInstance{
			{ID: "sonarr-1", Name: "Sonarr", URL: "http://localhost:8989", APIKey: "key1"},
			{ID: "sonarr-2", Name: "Sonarr 4K", URL: "http://localhost:8990", APIKey: "key2"},
		},
	})

	inst, ok := GetSonarrInstance("sonarr-2")
	if !ok {
		t.Fatal("expected to find sonarr-2")
	}
	if inst.Name != "Sonarr 4K" {
		t.Fatalf("expected Sonarr 4K, got %s", inst.Name)
	}

	_, ok = GetSonarrInstance("doesnt-exist")
	if ok {
		t.Fatal("expected not found for missing instance")
	}
}
