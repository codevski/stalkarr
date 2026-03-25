package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"stalkarr/internal/api"
	"stalkarr/internal/config"
	"stalkarr/internal/jobs"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/config"
	}

	if err := config.Init(dataDir); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	stalkerCtx, stalkerCancel := context.WithCancel(context.Background())
	defer stalkerCancel()
	stalker := jobs.NewStalkerJob(config.Get)
	go stalker.Start(stalkerCtx)

	r := api.NewRouter(stalker)
	r.SetTrustedProxies(nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Stalkarr running on :%s (data: %s)", port, dataDir)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stalkerCancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("Stalkarr stopped")
}
