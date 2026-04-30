package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sleeparr/internal/config"

	"github.com/gin-gonic/gin"
)

func maskKey(key string) string {
	if len(key) < 4 {
		return "••••••••"
	}
	return "••••••••" + key[len(key)-4:]
}

func generateID() string {
	return fmt.Sprintf("%d", len(config.Get().Sonarr)+1)
}

type instanceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	APIKeySet  bool   `json:"api_key_set"`
	APIKeyHint string `json:"api_key_hint"`
}

func toResponse(inst config.SonarrInstance) instanceResponse {
	return instanceResponse{
		ID:         inst.ID,
		Name:       inst.Name,
		URL:        inst.URL,
		APIKeySet:  inst.APIKey != "",
		APIKeyHint: maskKey(inst.APIKey),
	}
}

func getSettings(c *gin.Context) {
	cfg := config.Get()
	instances := make([]instanceResponse, len(cfg.Sonarr))
	for i, inst := range cfg.Sonarr {
		instances[i] = toResponse(inst)
	}
	c.JSON(http.StatusOK, gin.H{"sonarr": instances})
}

type instanceRequest struct {
	Name   string `json:"name" binding:"required"`
	URL    string `json:"url"  binding:"required"`
	APIKey string `json:"api_key"`
}

func addSonarrInstance(c *gin.Context) {
	var req instanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and url required"})
		return
	}

	cfg := config.Get()

	id := fmt.Sprintf("sonarr-%d", len(cfg.Sonarr)+1)

	inst := config.SonarrInstance{
		ID:     id,
		Name:   req.Name,
		URL:    strings.TrimRight(req.URL, "/"),
		APIKey: req.APIKey,
	}

	cfg.Sonarr = append(cfg.Sonarr, inst)
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save"})
		return
	}

	c.JSON(http.StatusOK, toResponse(inst))
}

func updateSonarrInstance(c *gin.Context) {
	id := c.Param("id")
	var req instanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and url required"})
		return
	}

	cfg := config.Get()
	found := false
	for i, inst := range cfg.Sonarr {
		if inst.ID == id {
			cfg.Sonarr[i].Name = req.Name
			cfg.Sonarr[i].URL = strings.TrimRight(req.URL, "/")
			if req.APIKey != "" {
				cfg.Sonarr[i].APIKey = req.APIKey
			}
			found = true
			if err := config.Save(cfg); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save"})
				return
			}
			c.JSON(http.StatusOK, toResponse(cfg.Sonarr[i]))
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
	}
}

func deleteSonarrInstance(c *gin.Context) {
	id := c.Param("id")
	cfg := config.Get()

	newInstances := cfg.Sonarr[:0]
	found := false
	for _, inst := range cfg.Sonarr {
		if inst.ID == id {
			found = true
			continue
		}
		newInstances = append(newInstances, inst)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	cfg.Sonarr = newInstances
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// TODO: Should be able to test before adding a instance
func testSonarrInstance(c *gin.Context) {
	id := c.Param("id")
	cfg := config.Get()

	var instance *config.SonarrInstance
	for i := range cfg.Sonarr {
		if cfg.Sonarr[i].ID == id {
			instance = &cfg.Sonarr[i]
			break
		}
	}

	if instance == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	url := strings.TrimRight(instance.URL, "/") + "/api/v3/system/status"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid URL", "ok": false})
		return
	}
	req.Header.Set("X-Api-Key", instance.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": "could not reach Sonarr — check the URL is correct and Sonarr is running",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": "connected but API key was rejected",
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": fmt.Sprintf("Sonarr returned unexpected status %d", resp.StatusCode),
		})
		return
	}

	var status struct {
		Version string `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&status)

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"version": status.Version,
	})
}

func getAgentSettings(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, cfg.Agent)
}

func (h *Handler) saveAgentSettings(c *gin.Context) {
	var req config.AgentConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.IntervalMinutes < 1 {
		req.IntervalMinutes = 1
	}
	if req.EpisodesPerRun < 1 {
		req.EpisodesPerRun = 1
	}
	if req.CooldownHours < 1 {
		req.CooldownHours = 1
	}
	cfg := config.Get()
	cfg.Agent = req
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save"})
		return
	}

	h.agent.Reload(h.appCtx)

	c.JSON(http.StatusOK, cfg.Agent)
}
