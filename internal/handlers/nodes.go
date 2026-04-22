package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nodestral/backend/internal/models"
)

type NodeHandler struct {
	db *sql.DB
}

func NewNodeHandler(db *sql.DB) *NodeHandler {
	return &NodeHandler{db: db}
}

func (h *NodeHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := h.db.Query(
		`SELECT id, COALESCE(user_id,'') as user_id, name, hostname, COALESCE(group_name,'') as group_name,
		        COALESCE(tags,'[]') as tags, os, COALESCE(kernel,'') as kernel, COALESCE(arch,'') as arch,
		        cpu_cores, ram_mb, disk_gb, COALESCE(provider,'') as provider, COALESCE(region,'') as region,
		        COALESCE(public_ip,'') as public_ip, COALESCE(private_ip,'') as private_ip,
		        status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE user_id = ? ORDER BY hostname`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var n models.Node
		var userIDStr, kernel, arch, provider, region, publicIP, privateIP, groupName, tags string
		var lastHB sql.NullTime
		if err := rows.Scan(&n.ID, &userIDStr, &n.Name, &n.Hostname, &groupName, &tags,
			&n.OS, &kernel, &arch, &n.CPUCores, &n.RAMMB, &n.DiskGB,
			&provider, &region, &publicIP, &privateIP,
			&n.Status, &lastHB, &n.CreatedAt, &n.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if userIDStr != "" {
			n.UserID = &userIDStr
		}
		n.Kernel = kernel
		n.Arch = arch
		n.Provider = provider
		n.Region = region
		n.PublicIP = publicIP
		n.PrivateIP = privateIP
		n.GroupName = groupName
		n.Tags = tags
		if lastHB.Valid {
			n.LastHeartbeat = &lastHB.Time
		}
		nodes = append(nodes, n)
	}
	if nodes == nil {
		nodes = []models.Node{}
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (h *NodeHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Param("id")

	var n models.Node
	var userIDStr, kernel, arch, provider, region, publicIP, privateIP, groupName, tags string
	var lastHB sql.NullTime
	err := h.db.QueryRow(
		`SELECT id, COALESCE(user_id,'') as user_id, name, hostname, COALESCE(group_name,'') as group_name,
		        COALESCE(tags,'[]') as tags, os, COALESCE(kernel,'') as kernel, COALESCE(arch,'') as arch,
		        cpu_cores, ram_mb, disk_gb, COALESCE(provider,'') as provider, COALESCE(region,'') as region,
		        COALESCE(public_ip,'') as public_ip, COALESCE(private_ip,'') as private_ip,
		        status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE id = ? AND user_id = ?`, nodeID, userID,
	).Scan(&n.ID, &userIDStr, &n.Name, &n.Hostname, &groupName, &tags,
		&n.OS, &kernel, &arch, &n.CPUCores, &n.RAMMB, &n.DiskGB,
		&provider, &region, &publicIP, &privateIP,
		&n.Status, &lastHB, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if userIDStr != "" {
		n.UserID = &userIDStr
	}
	n.Kernel = kernel
	n.Arch = arch
	n.Provider = provider
	n.Region = region
	n.PublicIP = publicIP
	n.PrivateIP = privateIP
	n.GroupName = groupName
	n.Tags = tags
	if lastHB.Valid {
		n.LastHeartbeat = &lastHB.Time
	}

	// Latest discovery
	var disc models.NodeDiscovery
	discErr := h.db.QueryRow(
		"SELECT id, node_id, discovered_at, services, packages, containers, ports, certificates, firewall, updates, ssh_users, monitoring_tools FROM node_discovery WHERE node_id = ? ORDER BY discovered_at DESC LIMIT 1",
		nodeID,
	).Scan(&disc.ID, &disc.NodeID, &disc.DiscoveredAt, &disc.Services, &disc.Packages, &disc.Containers,
		&disc.Ports, &disc.Certificates, &disc.Firewall, &disc.Updates, &disc.SSHUsers, &disc.MonitoringTools)

	// Latest metrics
	var metrics models.NodeMetrics
	metErr := h.db.QueryRow(
		"SELECT id, node_id, time, cpu_percent, ram_percent, ram_used_mb, disk_percent, disk_used_gb, net_rx_bytes, net_tx_bytes, load_1m, load_5m FROM node_metrics WHERE node_id = ? ORDER BY time DESC LIMIT 1",
		nodeID,
	).Scan(&metrics.ID, &metrics.NodeID, &metrics.Time, &metrics.CPUPercent, &metrics.RAMPercent, &metrics.RAMUsedMB,
		&metrics.DiskPercent, &metrics.DiskUsedGB, &metrics.NetRxBytes, &metrics.NetTxBytes, &metrics.Load1m, &metrics.Load5m)

	resp := gin.H{"node": n}
	if discErr == nil {
		resp["discovery"] = disc
	}
	if metErr == nil {
		resp["metrics"] = metrics
	}
	c.JSON(http.StatusOK, resp)
}

func (h *NodeHandler) GetMetrics(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Param("id")

	var count int
	h.db.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ? AND user_id = ?", nodeID, userID).Scan(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	duration := c.DefaultQuery("duration", "1h")
	interval := durationToInterval(duration)

	rows, err := h.db.Query(
		`SELECT id, node_id, time, cpu_percent, ram_percent, ram_used_mb, disk_percent, disk_used_gb,
		        net_rx_bytes, net_tx_bytes, load_1m, load_5m
		 FROM node_metrics WHERE node_id = ? AND time > datetime('now', ?) ORDER BY time DESC`, nodeID, "-"+interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var metrics []models.NodeMetrics
	for rows.Next() {
		var m models.NodeMetrics
		if err := rows.Scan(&m.ID, &m.NodeID, &m.Time, &m.CPUPercent, &m.RAMPercent, &m.RAMUsedMB,
			&m.DiskPercent, &m.DiskUsedGB, &m.NetRxBytes, &m.NetTxBytes, &m.Load1m, &m.Load5m); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}
	if metrics == nil {
		metrics = []models.NodeMetrics{}
	}
	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

func (h *NodeHandler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Param("id")

	var req struct {
		Name      string   `json:"name"`
		GroupName string   `json:"group_name"`
		Tags      []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int
	h.db.QueryRow("SELECT COUNT(*) FROM nodes WHERE id = ? AND user_id = ?", nodeID, userID).Scan(&count)
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	tagsJSON, _ := json.Marshal(req.Tags)
	now := time.Now()
	_, err := h.db.Exec(
		"UPDATE nodes SET name = COALESCE(NULLIF(?,''), name), group_name = COALESCE(NULLIF(?,''), group_name), tags = COALESCE(?, tags), updated_at = ? WHERE id = ?",
		req.Name, req.GroupName, tagsJSON, now, nodeID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *NodeHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Param("id")

	result, err := h.db.Exec("DELETE FROM nodes WHERE id = ? AND user_id = ?", nodeID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *NodeHandler) ListUnclaimed(c *gin.Context) {
	rows, err := h.db.Query(
		`SELECT id, COALESCE(user_id,'') as user_id, name, hostname, COALESCE(group_name,'') as group_name,
		        COALESCE(tags,'[]') as tags, os, COALESCE(kernel,'') as kernel, COALESCE(arch,'') as arch,
		        cpu_cores, ram_mb, disk_gb, COALESCE(provider,'') as provider, COALESCE(region,'') as region,
		        COALESCE(public_ip,'') as public_ip, COALESCE(private_ip,'') as private_ip,
		        status, last_heartbeat, created_at, updated_at
		 FROM nodes WHERE user_id IS NULL ORDER BY hostname`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var n models.Node
		var userIDStr, kernel, arch, provider, region, publicIP, privateIP, groupName, tags string
		var lastHB sql.NullTime
		if err := rows.Scan(&n.ID, &userIDStr, &n.Name, &n.Hostname, &groupName, &tags,
			&n.OS, &kernel, &arch, &n.CPUCores, &n.RAMMB, &n.DiskGB,
			&provider, &region, &publicIP, &privateIP,
			&n.Status, &lastHB, &n.CreatedAt, &n.UpdatedAt); err != nil {
			continue
		}
		n.Kernel = kernel
		n.Arch = arch
		n.Provider = provider
		n.Region = region
		n.PublicIP = publicIP
		n.PrivateIP = privateIP
		n.GroupName = groupName
		n.Tags = tags
		if lastHB.Valid {
			n.LastHeartbeat = &lastHB.Time
		}
		nodes = append(nodes, n)
	}
	if nodes == nil {
		nodes = []models.Node{}
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (h *NodeHandler) Claim(c *gin.Context) {
	userID := c.GetString("user_id")
	nodeID := c.Param("id")

	result, err := h.db.Exec(
		"UPDATE nodes SET user_id = ?, updated_at = ? WHERE id = ? AND user_id IS NULL",
		userID, time.Now(), nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "node already claimed or not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "node claimed"})
}

func (h *NodeHandler) AgentRegister(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nodeID := uuid.New().String()
	authToken := uuid.New().String()
	tagsJSON, _ := json.Marshal([]string{})
	now := time.Now()

	_, err := h.db.Exec(
		`INSERT INTO nodes (id, user_id, name, hostname, os, kernel, arch, cpu_cores, ram_mb, disk_gb,
		                    provider, region, public_ip, private_ip, auth_token, status, tags, last_heartbeat, created_at, updated_at)
		 VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'online', ?, ?, ?, ?)`,
		nodeID, req.System.Hostname, req.System.Hostname, req.System.OS, req.System.Kernel,
		req.System.Arch, req.System.CPUCores, req.System.RAMMB, req.System.DiskGB,
		req.Provider.Name, req.Provider.Region, req.System.PublicIP, req.System.PrivateIP,
		authToken, tagsJSON, now, now, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register node", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.RegisterResponse{NodeID: nodeID, AuthToken: authToken})
}

func (h *NodeHandler) AgentHeartbeat(c *gin.Context) {
	var req models.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var authToken string
	err := h.db.QueryRow("SELECT auth_token FROM nodes WHERE id = ?", req.NodeID).Scan(&authToken)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown node"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if authToken != c.GetString("agent_token") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	now := time.Now()
	_, err = h.db.Exec(
		`INSERT INTO node_metrics (node_id, time, cpu_percent, ram_percent, ram_used_mb, disk_percent, disk_used_gb, net_rx_bytes, net_tx_bytes, load_1m, load_5m)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.NodeID, now, req.CPUPercent, req.RAMPercent, req.RAMUsedMB,
		req.DiskPercent, req.DiskUsedGB, req.NetRxBytes, req.NetTxBytes, req.Load1m, req.Load5m,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store metrics"})
		return
	}

	_, err = h.db.Exec("UPDATE nodes SET status = 'online', last_heartbeat = ?, updated_at = ? WHERE id = ?", now, now, req.NodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *NodeHandler) AgentDiscovery(c *gin.Context) {
	var body struct {
		NodeID          string `json:"node_id"`
		Services        string `json:"services"`
		Packages        string `json:"packages"`
		Containers      string `json:"containers"`
		Ports           string `json:"listening_ports"`
		Certificates    string `json:"certificates"`
		Firewall        string `json:"firewall"`
		Updates         string `json:"updates"`
		SSHUsers        string `json:"ssh_users"`
		MonitoringTools string `json:"monitoring_tools"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
		return
	}

	now := time.Now()
	_, err := h.db.Exec(
		`INSERT INTO node_discovery (node_id, discovered_at, services, packages, containers, ports, certificates, firewall, updates, ssh_users, monitoring_tools)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		body.NodeID, now, body.Services, body.Packages, body.Containers, body.Ports,
		body.Certificates, body.Firewall, body.Updates, body.SSHUsers, body.MonitoringTools,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store discovery"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func durationToInterval(d string) string {
	switch d {
	case "5m", "15m", "30m":
		return "5 minutes"
	case "1h":
		return "1 hour"
	case "6h":
		return "6 hours"
	case "24h", "1d":
		return "1 day"
	case "7d":
		return "7 days"
	default:
		return "1 hour"
	}
}
