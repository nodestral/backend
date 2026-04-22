package models

import "time"

// User represents a registered user.
type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Plan         string    `json:"plan" db:"plan"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Node represents a registered server.
type Node struct {
	ID            string     `json:"id" db:"id"`
	UserID        *string    `json:"user_id" db:"user_id"`
	Name          string     `json:"name" db:"name"`
	Hostname      string     `json:"hostname" db:"hostname"`
	GroupName     string     `json:"group_name" db:"group_name"`
	Tags          string     `json:"tags" db:"tags"`
	OS            string     `json:"os" db:"os"`
	Kernel        string     `json:"kernel" db:"kernel"`
	Arch          string     `json:"arch" db:"arch"`
	CPUCores      int        `json:"cpu_cores" db:"cpu_cores"`
	RAMMB         int64      `json:"ram_mb" db:"ram_mb"`
	DiskGB        int64      `json:"disk_gb" db:"disk_gb"`
	Provider      string     `json:"provider" db:"provider"`
	Region        string     `json:"region" db:"region"`
	PublicIP      string     `json:"public_ip" db:"public_ip"`
	PrivateIP     string     `json:"private_ip" db:"private_ip"`
	AuthToken     string     `json:"-" db:"auth_token"`
	Status        string     `json:"status" db:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// NodeMetrics represents a heartbeat metric snapshot.
type NodeMetrics struct {
	ID          int64     `json:"id" db:"id"`
	NodeID      string    `json:"node_id" db:"node_id"`
	Time        time.Time `json:"time" db:"time"`
	CPUPercent  float64   `json:"cpu_percent" db:"cpu_percent"`
	RAMPercent  float64   `json:"ram_percent" db:"ram_percent"`
	RAMUsedMB   int64     `json:"ram_used_mb" db:"ram_used_mb"`
	DiskPercent float64   `json:"disk_percent" db:"disk_percent"`
	DiskUsedGB  int64     `json:"disk_used_gb" db:"disk_used_gb"`
	NetRxBytes  int64     `json:"net_rx_bytes" db:"net_rx_bytes"`
	NetTxBytes  int64     `json:"net_tx_bytes" db:"net_tx_bytes"`
	Load1m      float64   `json:"load_1m" db:"load_1m"`
	Load5m      float64   `json:"load_5m" db:"load_5m"`
}

// NodeDiscovery represents a system discovery snapshot.
type NodeDiscovery struct {
	ID              int64     `json:"id" db:"id"`
	NodeID          string    `json:"node_id" db:"node_id"`
	DiscoveredAt    time.Time `json:"discovered_at" db:"discovered_at"`
	Services        string    `json:"services" db:"services"`
	Packages        string    `json:"packages" db:"packages"`
	Containers      string    `json:"containers" db:"containers"`
	Ports           string    `json:"ports" db:"ports"`
	Certificates    string    `json:"certificates" db:"certificates"`
	Firewall        string    `json:"firewall" db:"firewall"`
	Updates         string    `json:"updates" db:"updates"`
	SSHUsers        string    `json:"ssh_users" db:"ssh_users"`
	MonitoringTools string    `json:"monitoring_tools" db:"monitoring_tools"`
}

// RegisterRequest is sent by the agent on first registration.
type RegisterRequest struct {
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

// RegisterResponse is returned after agent registration.
type RegisterResponse struct {
	NodeID    string `json:"node_id"`
	AuthToken string `json:"auth_token"`
}

// HeartbeatRequest is sent by the agent every 30s.
type HeartbeatRequest struct {
	NodeID      string  `json:"node_id"`
	CPUPercent  float64 `json:"cpu_percent"`
	RAMPercent  float64 `json:"ram_percent"`
	RAMUsedMB   int64   `json:"ram_used_mb"`
	DiskPercent float64 `json:"disk_percent"`
	DiskUsedGB  int64   `json:"disk_used_gb"`
	NetRxBytes  int64   `json:"net_rx_bytes"`
	NetTxBytes  int64   `json:"net_tx_bytes"`
	Load1m      float64 `json:"load_1m"`
	Load5m      float64 `json:"load_5m"`
}

// AuthRequest for login/register.
type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// AuthResponse returned after login/register.
type AuthResponse struct {
	Token string  `json:"token"`
	User  User    `json:"user"`
}
