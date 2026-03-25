package api

import (
	"fmt"
	"io/fs"
	"net/http"

	"stalkarr/internal/arr"
	"stalkarr/internal/config"
	"stalkarr/internal/static"
	"stalkarr/internal/version"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Authorization", "Content-Type"},
	}))

	r.Use(securityHeaders())

	// API routes
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
		protected.GET("/dashboard", getDashboard)
		protected.GET("/version", getVersion)
		protected.POST("/settings/sonarr/:id/test", testSonarrInstance)
		protected.POST("/auth/logout", handleLogout)
		protected.POST("/auth/password", handleChangePassword)

		sonarr := protected.Group("/sonarr/:id")
		{
			sonarr.GET("/missing", getMissing)
			sonarr.POST("/hunt", triggerHunt)
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

func triggerHunt(c *gin.Context) {
	_, ok := getSonarrClient(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "hunting"})
}

func getDashboard(c *gin.Context) {
	cfg := config.Get()

	type instanceSummary struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		MissingCount int    `json:"missingCount"`
		Error        string `json:"error,omitempty"`
	}

	summaries := make([]instanceSummary, 0, len(cfg.Sonarr))
	for _, inst := range cfg.Sonarr {
		client := arr.NewSonarrClient(inst.URL, inst.APIKey)
		result, err := client.GetMissingEpisodes(1, 1, "")
		if err != nil {
			summaries = append(summaries, instanceSummary{
				ID: inst.ID, Name: inst.Name, Error: "unreachable",
			})
			continue
		}
		summaries = append(summaries, instanceSummary{
			ID: inst.ID, Name: inst.Name, MissingCount: result.TotalCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"sonarr": summaries})
}

func getVersion(c *gin.Context) {
	latest, hasUpdate, _ := version.CheckForUpdate()
	c.JSON(http.StatusOK, gin.H{
		"current":   version.Version,
		"latest":    latest,
		"hasUpdate": hasUpdate,
	})
}
