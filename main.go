// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/kirant400/rp-wrapper/cache"
	"github.com/kirant400/rp-wrapper/config"
	"github.com/kirant400/rp-wrapper/handlers"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	configPath := flag.String("config", "config.yaml", "Path to config YAML file")
	flag.Parse()

	// Load configuration
	config.Load(*configPath)

	// Override port from env if present (common in Docker)
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := fmt.Sscanf(envPort, "%d", port); err == nil && p == 1 {
			log.Printf("Port overridden by PORT env: %d", *port)
		}
	}

	// Initialize Redis
	cache.Init()

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Register all endpoints from config
	registerEndpoints(r)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("rp-wrapper starting on http://localhost%s", addr)
	log.Printf("Config loaded from: %s", *configPath)

	// Graceful shutdown
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Optional: close Redis
	cache.Close()

	log.Println("rp-wrapper stopped")
}

func registerEndpoints(r *gin.Engine) {
	cfg := config.Global.Endpoints

	// Data filling endpoints (always from legacy → Redis)
	if cfg.Data.Enabled {
		method := "POST"
		if cfg.Data.Method != "" {
			method = cfg.Data.Method
		}
		p := cfg.Data.Paths

		r.Handle(method, p["departments"], handlers.FillDepartments)
		r.Handle(method, p["areas"], handlers.FillAreas)
		r.Handle(method, p["positions"], handlers.FillPositions)
		r.Handle(method, p["employees"], handlers.FillEmployees)
		r.Handle(method, p["transactions"], handlers.FillTransactions)
		r.Handle(method, p["transactions"]+"/:endtime", handlers.FillTransactions)
	}

	// Attendance endpoints
	if cfg.Attendance.Enabled {
		method := "POST"
		if cfg.Attendance.Method != "" {
			method = cfg.Attendance.Method
		}
		p := cfg.Attendance.Paths

		r.Handle(method, p["get"], handlers.GetAttendance)
		r.Handle(method, p["set"], handlers.SetAttendance)
		r.Handle(method, p["reset"], handlers.ResetAttendance)
		if display, ok := p["display"]; ok {
			r.Handle(method, display, handlers.DisplayAttendanceMapping)
		}
	}

	// Reset endpoints
	if cfg.Reset.Enabled {
		method := "POST"
		if cfg.Reset.Method != "" {
			method = cfg.Reset.Method
		}
		p := cfg.Reset.Paths

		if path, ok := p["mapping"]; ok {
			r.Handle(method, path, handlers.ResetMapping)
		}
		if path, ok := p["departments"]; ok {
			r.Handle(method, path, handlers.ResetDepartments)
		}
		if path, ok := p["areas"]; ok {
			r.Handle(method, path, handlers.ResetAreas)
		}
		if path, ok := p["positions"]; ok {
			r.Handle(method, path, handlers.ResetPositions)
		}
		if path, ok := p["employees"]; ok {
			r.Handle(method, path, handlers.ResetEmployees)
		}
	}

	// Mapping upload
	if cfg.MappingUpload.Enabled {
		method := "POST"
		if cfg.MappingUpload.Method != "" {
			method = cfg.MappingUpload.Method
		}
		r.Handle(method, cfg.MappingUpload.Path, handlers.UploadMapping)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "rp-wrapper"})
	})
}