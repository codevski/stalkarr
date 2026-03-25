package main

import (
	"log"
	"os"
	"stalkarr/internal/api"
	"stalkarr/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./config"
	}

	if err := config.Init(dataDir); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := api.NewRouter()
	r.SetTrustedProxies(nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Stalkarr running on :%s (data: %s)", port, dataDir)
	r.Run(":" + port)
}
