package api

import (
	"fmt"
	"io/fs"
	"net/http"
	"stalkarr/internal/arr"
	"stalkarr/internal/config"
	"stalkarr/internal/jobs"
	"stalkarr/internal/static"
	"stalkarr/internal/version"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	stalker *jobs.StalkerJob
}

func NewRouter(stalker *jobs.StalkerJob) *gin.Engine {
	h := &Handler{stalker: stalker}

	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))
	r.Use(securityHeaders())

	public := r.Group("/api")
	public.Use(rateLimiter())
	{
		public.POST("/login", strictRateLimiter(), handleLogin)
		public.POST("/setup", strictRateLimiter(), handleSetupUser)
		public.POST("/auth/refresh", handleRefresh)
	}

	protected := r.Group("/api")
	protected.Use(rateLimiter())
	protected.Use(authRequired())
	{
		protected.GET("/settings", getSettings)
		protected.POST("/settings/sonarr", addSonarrInstance)
		protected.PUT("/settings/sonarr/:id", updateSonarrInstance)
		protected.DELETE("/settings/sonarr/:id", deleteSonarrInstance)
		protected.POST("/settings/sonarr/:id/test", testSonarrInstance)
		protected.GET("/settings/stalk", getStalkSettings)
		protected.POST("/settings/stalk", saveStalkSettings)
		protected.GET("/dashboard", h.getDashboard)
		protected.GET("/version", getVersion)
		protected.POST("/auth/logout", handleLogout)
		protected.POST("/auth/password", handleChangePassword)
		protected.GET("/jobs/status", h.getJobStatus)

		sonarr := protected.Group("/sonarr/:id")
		{
			sonarr.GET("/missing", getMissing)
			sonarr.POST("/stalk", h.stalkEpisodes)
			sonarr.POST("/stalk/all", h.stalkAll)
		}
	}

	assets := static.Assets()
	r.GET("/", func(c *gin.Context) {
		index, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path[1:]
		f, err := assets.Open(path)
		if err == nil {
			f.Close()
			c.FileFromFS(path, http.FS(assets))
			return
		}
		index, err := fs.ReadFile(assets, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", index)
	})

	return r
}

func getSonarrClient(c *gin.Context) (*arr.SonarrClient, bool) {
	id := c.Param("id")
	instance, ok := config.GetSonarrInstance(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return nil, false
	}
	return arr.NewSonarrClient(instance.URL, instance.APIKey), true
}

func getMissing(c *gin.Context) {
	client, ok := getSonarrClient(c)
	if !ok {
		return
	}
	page, pageSize := 1, 20
	search := c.Query("search")
	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("pageSize"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}
	result, err := client.GetMissingEpisodes(page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) stalkEpisodes(c *gin.Context) {
	client, ok := getSonarrClient(c)
	if !ok {
		return
	}

	var req struct {
		EpisodeIDs []int `json:"episodeIds" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "episodeIds is required and must be non-empty"})
		return
	}

	result, err := client.TriggerEpisodeSearch(req.EpisodeIDs)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.stalker.RecordManualStalk(c.Param("id"), len(req.EpisodeIDs))

	c.JSON(http.StatusOK, gin.H{
		"message":   result.Message,
		"commandId": result.CommandID,
		"count":     len(req.EpisodeIDs),
	})
}

func (h *Handler) getJobStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.stalker.Status())
}

func (h *Handler) getDashboard(c *gin.Context) {
	cfg := config.Get()
	jobStatus := h.stalker.Status()

	type instanceSummary struct {
		ID           string     `json:"id"`
		Name         string     `json:"name"`
		MissingCount int        `json:"missingCount"`
		LastStalk    *time.Time `json:"lastStalk"`
		LastCount    int        `json:"lastStalkCount"`
		State        string     `json:"state"`
		Error        string     `json:"error,omitempty"`
	}

	stalksToday := 0
	summaries := make([]instanceSummary, 0, len(cfg.Sonarr))
	for _, inst := range cfg.Sonarr {
		client := arr.NewSonarrClient(inst.URL, inst.APIKey)
		result, err := client.GetMissingEpisodes(1, 1, "")

		status := jobStatus.Instances[inst.ID]
		stalksToday += status.LastCount

		if err != nil {
			summaries = append(summaries, instanceSummary{
				ID: inst.ID, Name: inst.Name, State: "error", Error: "unreachable",
			})
			continue
		}
		summaries = append(summaries, instanceSummary{
			ID:           inst.ID,
			Name:         inst.Name,
			MissingCount: result.TotalCount,
			LastStalk:    status.LastRun,
			LastCount:    status.LastCount,
			State:        status.State,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sonarr":      summaries,
		"stalksToday": stalksToday,
	})
}

func getVersion(c *gin.Context) {
	latest, hasUpdate, _ := version.CheckForUpdate()
	c.JSON(http.StatusOK, gin.H{
		"current":   version.Version,
		"latest":    latest,
		"hasUpdate": hasUpdate,
	})
}

func (h *Handler) stalkAll(c *gin.Context) {
	client, ok := getSonarrClient(c)
	if !ok {
		return
	}

	result, err := client.GetMissingEpisodes(1, 1000, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	if len(result.Episodes) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No missing episodes to stalk", "count": 0})
		return
	}

	ids := make([]int, len(result.Episodes))
	for i, ep := range result.Episodes {
		ids[i] = ep.ID
	}

	stalkResult, err := client.TriggerEpisodeSearch(ids)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   stalkResult.Message,
		"commandId": stalkResult.CommandID,
		"count":     len(ids),
	})
}
