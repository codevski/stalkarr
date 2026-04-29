package api

import (
	"fmt"
	"io/fs"
	"net/http"
	"sleeparr/internal/arr"
	"sleeparr/internal/config"
	"sleeparr/internal/jobs"
	"sleeparr/internal/static"
	"sleeparr/internal/version"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	agent *jobs.AgentJob
}

func NewRouter(agent *jobs.AgentJob) *gin.Engine {
	h := &Handler{agent: agent}

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
		public.GET("/setup/status", handleSetupStatus)
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
		protected.GET("/settings/agent", getAgentSettings)
		protected.POST("/settings/agent", saveAgentSettings)
		protected.GET("/dashboard", h.getDashboard)
		protected.GET("/version", getVersion)
		protected.POST("/auth/logout", handleLogout)
		protected.POST("/auth/password", handleChangePassword)
		protected.GET("/jobs/status", h.getJobStatus)

		sonarr := protected.Group("/sonarr/:id")
		{
			sonarr.GET("/missing", getMissing)
			sonarr.POST("/run", h.runEpisodes)
			sonarr.POST("/run/all", h.runAll)
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

func (h *Handler) runEpisodes(c *gin.Context) {
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

	h.agent.RecordManualRun(c.Param("id"), len(req.EpisodeIDs))

	c.JSON(http.StatusOK, gin.H{
		"message":   result.Message,
		"commandId": result.CommandID,
		"count":     len(req.EpisodeIDs),
	})
}

func (h *Handler) getJobStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.agent.Status())
}

func (h *Handler) getDashboard(c *gin.Context) {
	cfg := config.Get()
	jobStatus := h.agent.Status()

	type instanceSummary struct {
		ID           string     `json:"id"`
		Name         string     `json:"name"`
		MissingCount int        `json:"missingCount"`
		LastRun      *time.Time `json:"lastRun"`
		LastCount    int        `json:"lastRunCount"`
		State        string     `json:"state"`
		Error        string     `json:"error,omitempty"`
	}

	runsToday := 0
	summaries := make([]instanceSummary, 0, len(cfg.Sonarr))
	for _, inst := range cfg.Sonarr {
		client := arr.NewSonarrClient(inst.URL, inst.APIKey)
		result, err := client.GetMissingEpisodes(1, 1, "")

		status := jobStatus.Instances[inst.ID]
		runsToday += status.LastCount

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
			LastRun:      status.LastRun,
			LastCount:    status.LastCount,
			State:        status.State,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sonarr":    summaries,
		"runsToday": runsToday,
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

func (h *Handler) runAll(c *gin.Context) {
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
		c.JSON(http.StatusOK, gin.H{"message": "No missing episodes to find", "count": 0})
		return
	}

	ids := make([]int, len(result.Episodes))
	for i, ep := range result.Episodes {
		ids[i] = ep.ID
	}

	runResult, err := client.TriggerEpisodeSearch(ids)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   runResult.Message,
		"commandId": runResult.CommandID,
		"count":     len(ids),
	})
}
