# Agent Permissions Guide

The Nodestral agent is designed to run with **minimal privileges** by default. Discovery features that require elevated access are **disabled by default** and must be explicitly enabled by the user.

## Default Behavior (No Special Permissions)

These features work out of the box with a standard unprivileged user:

| Feature | What It Collects | How |
|---------|-----------------|-----|
| **Heartbeat** | CPU, RAM, disk, network, load | `/proc/*`, syscall |
| **Services** | Running systemd services | `systemctl list-units` (public info) |
| **Packages** | Notable installed packages + versions | `dpkg-query` (public info) |
| **Ports** | Listening TCP ports | `/proc/net/tcp`, `/proc/net/tcp6` |
| **SSH Users** | Users with login shells | `/etc/passwd` (world-readable) |
| **Monitoring** | Detected monitoring tools | `which`/`systemctl is-active` |

**No special permissions needed.** The agent works immediately after install.

---

## Opt-In Features (Require Additional Permissions)

These features are **disabled by default**. Enable them in `/etc/nodestral/agent.yaml`:

```yaml
discovery:
  containers: true      # Docker containers
  certificates: true    # SSL/TLS certificates
  firewall: true        # Firewall status and rules
  os_updates: true      # Pending OS security updates
```

### containers: true — Docker Containers

```bash
sudo usermod -aG docker nodestral
```

### certificates: true — SSL/TLS Certificates

```bash
sudo chmod 755 /etc/letsencrypt/live /etc/letsencrypt/archive
```

### firewall: true — Firewall Status

```bash
echo "nodestral ALL=(root) NOPASSWD: /usr/sbin/ufw status, /usr/sbin/iptables -L -n" | sudo tee /etc/sudoers.d/nodestral-firewall
sudo chmod 440 /etc/sudoers.d/nodestral-firewall
```

### os_updates: true — Pending OS Updates

```bash
echo "nodestral ALL=(root) NOPASSWD: /usr/bin/apt list --upgradable" | sudo tee /etc/sudoers.d/nodestral-updates
sudo chmod 440 /etc/sudoers.d/nodestral-updates
```

---

## What the Agent Will NEVER Do

- Read file contents (only metadata — certs, package versions)
- Execute arbitrary commands
- Modify system configuration
- Access user data or home directories
- Store credentials beyond its own config token
- Make outbound connections except to the configured API URL
