package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InstallTokenHandler struct {
	db *sql.DB
}

func NewInstallTokenHandler(db *sql.DB) *InstallTokenHandler {
	return &InstallTokenHandler{db: db}
}

func (h *InstallTokenHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		MaxUses  int       `json:"max_uses"`
		ExpiresAt *string   `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req.MaxUses = 2
	}
	if req.MaxUses <= 0 {
		req.MaxUses = 2
	}

	id := uuid.New().String()
	token := uuid.New().String()
	now := time.Now()

	var expiresAt sql.NullTime
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = sql.NullTime{Time: t, Valid: true}
		}
	}

	_, err := h.db.Exec(
		"INSERT INTO install_tokens (id, user_id, token, max_uses, uses, created_at, expires_at) VALUES (?, ?, ?, ?, 0, ?, ?)",
		id, userID, token, req.MaxUses, now, expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token"})
		return
	}

	resp := gin.H{
		"id":        id,
		"token":     token,
		"max_uses":  req.MaxUses,
		"uses":      0,
		"created_at": now,
	}
	if expiresAt.Valid {
		resp["expires_at"] = expiresAt.Time
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *InstallTokenHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := h.db.Query(
		"SELECT id, user_id, token, max_uses, uses, created_at, expires_at FROM install_tokens WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var tokens []gin.H
	for rows.Next() {
		var id, userID, token string
		var maxUses, uses int
		var createdAt time.Time
		var expiresAt sql.NullTime
		if err := rows.Scan(&id, &userID, &token, &maxUses, &uses, &createdAt, &expiresAt); err != nil {
			continue
		}
		t := gin.H{
			"id":         id,
			"token":      token,
			"max_uses":   maxUses,
			"uses":       uses,
			"created_at": createdAt,
		}
		if expiresAt.Valid {
			t["expires_at"] = expiresAt.Time
		}
		tokens = append(tokens, t)
	}
	if tokens == nil {
		tokens = []gin.H{}
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (h *InstallTokenHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	tokenID := c.Param("id")

	result, err := h.db.Exec("DELETE FROM install_tokens WHERE id = ? AND user_id = ?", tokenID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "token not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}

// AgentRegisterWithToken handles agent registration using an install token.
// The node is directly assigned to the token owner's account.
func (h *InstallTokenHandler) AgentRegisterWithToken(c *gin.Context) {
	tokenStr := c.GetHeader("X-Install-Token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Install-Token header required"})
		return
	}

	// Look up token
	var id, userID string
	var maxUses, uses int
	var expiresAt sql.NullTime
	err := h.db.QueryRow(
		"SELECT id, user_id, max_uses, uses, expires_at FROM install_tokens WHERE token = ?",
		tokenStr,
	).Scan(&id, &userID, &maxUses, &uses, &expiresAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid install token"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Check expiry
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "install token expired"})
		return
	}

	// Check usage
	if uses >= maxUses {
		c.JSON(http.StatusForbidden, gin.H{"error": "install token fully used"})
		return
	}

	// Parse registration body
	var req struct {
		System struct {
			Hostname  string `json:"hostname"`
			OS        string `json:"os"`
			Kernel    string `json:"kernel"`
			Arch      string `json:"arch"`
			CPUCores  int    `json:"cpu_cores"`
			RAMMB     int64  `json:"ram_mb"`
			DiskGB    int64  `json:"disk_gb"`
			PublicIP  string `json:"public_ip"`
			PrivateIP string `json:"private_ip"`
		} `json:"system"`
		Provider struct {
			Name   string `json:"name"`
			Region string `json:"region"`
		} `json:"provider"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nodeID := uuid.New().String()
	authToken := uuid.New().String()
	tagsJSON, _ := json.Marshal([]string{})
	now := time.Now()

	// Create node with user_id from token
	_, err = h.db.Exec(
		`INSERT INTO nodes (id, user_id, name, hostname, os, kernel, arch, cpu_cores, ram_mb, disk_gb,
		                    provider, region, public_ip, private_ip, auth_token, status, tags, last_heartbeat, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'online', ?, ?, ?, ?)`,
		nodeID, userID, req.System.Hostname, req.System.Hostname, req.System.OS, req.System.Kernel,
		req.System.Arch, req.System.CPUCores, req.System.RAMMB, req.System.DiskGB,
		req.Provider.Name, req.Provider.Region, req.System.PublicIP, req.System.PrivateIP,
		authToken, string(tagsJSON), now, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register node"})
		return
	}

	// Increment token usage
	h.db.Exec("UPDATE install_tokens SET uses = uses + 1 WHERE id = ?", id)

	c.JSON(http.StatusOK, gin.H{"node_id": nodeID, "auth_token": authToken})
}
