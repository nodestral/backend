package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nodestral/backend/internal/auth"
	"github.com/nodestral/backend/internal/handlers"
	"github.com/nodestral/backend/internal/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production"
		log.Println("WARNING: using default JWT_SECRET, set JWT_SECRET env var")
	}
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./nodestral.db"
	}
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	// Open SQLite
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	// Setup services
	authSvc := auth.NewService(jwtSecret)
	authHandler := handlers.NewAuthHandler(db, authSvc)
	nodeHandler := handlers.NewNodeHandler(db)

	// Router
	r := gin.Default()
	r.Use(middleware.CORSMiddleware(strings.Split(allowedOrigins, ",")))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Public routes
	api := r.Group("")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/agent/register", nodeHandler.AgentRegister)

		agentAuth := api.Group("/agent")
		agentAuth.Use(middleware.AgentAuthMiddleware())
		{
			agentAuth.POST("/heartbeat", nodeHandler.AgentHeartbeat)
			agentAuth.POST("/discovery", nodeHandler.AgentDiscovery)
		}
	}

	// Protected routes
	protected := r.Group("")
	protected.Use(middleware.AuthMiddleware(authSvc))
	{
		protected.GET("/nodes", nodeHandler.List)
		protected.GET("/nodes/unclaimed", nodeHandler.ListUnclaimed)
		protected.POST("/nodes/:id/claim", nodeHandler.Claim)
		protected.GET("/nodes/:id", nodeHandler.Get)
		protected.GET("/nodes/:id/metrics", nodeHandler.GetMetrics)
		protected.PATCH("/nodes/:id", nodeHandler.Update)
		protected.DELETE("/nodes/:id", nodeHandler.Delete)
	}

	fmt.Printf("Nodestral Backend starting on :%s\n", port)
	fmt.Printf("Database: %s\n", dbPath)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		plan TEXT DEFAULT 'free',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id TEXT PRIMARY KEY,
		user_id TEXT,
		name TEXT NOT NULL DEFAULT '',
		hostname TEXT NOT NULL DEFAULT '',
		group_name TEXT NOT NULL DEFAULT '',
		tags TEXT NOT NULL DEFAULT '[]',
		os TEXT NOT NULL DEFAULT '',
		kernel TEXT NOT NULL DEFAULT '',
		arch TEXT NOT NULL DEFAULT '',
		cpu_cores INTEGER NOT NULL DEFAULT 0,
		ram_mb INTEGER NOT NULL DEFAULT 0,
		disk_gb INTEGER NOT NULL DEFAULT 0,
		provider TEXT NOT NULL DEFAULT '',
		region TEXT NOT NULL DEFAULT '',
		public_ip TEXT NOT NULL DEFAULT '',
		private_ip TEXT NOT NULL DEFAULT '',
		auth_token TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'online',
		last_heartbeat DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		UNIQUE(auth_token)
	);

	CREATE TABLE IF NOT EXISTS node_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		time DATETIME NOT NULL,
		cpu_percent REAL NOT NULL DEFAULT 0,
		ram_percent REAL NOT NULL DEFAULT 0,
		ram_used_mb INTEGER NOT NULL DEFAULT 0,
		disk_percent REAL NOT NULL DEFAULT 0,
		disk_used_gb INTEGER NOT NULL DEFAULT 0,
		net_rx_bytes INTEGER NOT NULL DEFAULT 0,
		net_tx_bytes INTEGER NOT NULL DEFAULT 0,
		load_1m REAL NOT NULL DEFAULT 0,
		load_5m REAL NOT NULL DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_node_metrics_node_id ON node_metrics(node_id, time);

	CREATE TABLE IF NOT EXISTS node_discovery (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		discovered_at DATETIME NOT NULL,
		services TEXT NOT NULL DEFAULT '[]',
		packages TEXT NOT NULL DEFAULT '[]',
		containers TEXT NOT NULL DEFAULT '[]',
		ports TEXT NOT NULL DEFAULT '[]',
		certificates TEXT NOT NULL DEFAULT '[]',
		firewall TEXT NOT NULL DEFAULT '{}',
		updates TEXT NOT NULL DEFAULT '{}',
		ssh_users TEXT NOT NULL DEFAULT '[]',
		monitoring_tools TEXT NOT NULL DEFAULT '[]'
	);

	CREATE INDEX IF NOT EXISTS idx_node_discovery_node_id ON node_discovery(node_id);
	`
	_, err := db.Exec(schema)
	return err
}
