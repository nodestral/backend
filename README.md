# Nodestral Backend (Community Edition)

Self-hostable API server for Nodestral fleet management. Single binary, SQLite database, zero external dependencies.

## Quick Start

```bash
# Build
make build

# Run (creates nodestral.db automatically)
PORT=8080 JWT_SECRET=my-secret ./bin/nodestral-backend
```

## Configuration

| Env Variable | Default | Description |
|-------------|---------|-------------|
| `PORT` | `8080` | Server port |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `DATABASE_URL` | `./nodestral.db` | SQLite database path |
| `ALLOWED_ORIGINS` | `*` | Comma-separated CORS origins |

## API Endpoints

### Auth
```
POST /auth/register    { email, password }  → { token, user }
POST /auth/login       { email, password }  → { token, user }
```

### Agent (no auth required)
```
POST /agent/register   { system, provider }  → { node_id, auth_token }
POST /agent/heartbeat  (Bearer agent token)
POST /agent/discovery  (Bearer agent token)
```

### Nodes (user auth required)
```
GET    /nodes              → { nodes }
GET    /nodes/unclaimed    → { nodes }
POST   /nodes/:id/claim    → { message }
GET    /nodes/:id          → { node, discovery, metrics }
GET    /nodes/:id/metrics  → { metrics }
PATCH  /nodes/:id          → { message }
DELETE /nodes/:id          → { message }
```

### Health
```
GET /health → { status: "ok" }
```

## What's Included

- Agent registration and heartbeat
- JWT authentication (register/login)
- Node management (list, detail, update, delete)
- Node claiming (agent registers unclaimed, user claims)
- Metrics storage and retrieval
- System discovery snapshots
- CORS middleware

## What's Not Included (SaaS-only features)

These features are only available in the [hosted Nodestral platform](https://nodestral.web.id):

- Billing and subscription management
- Node limit enforcement (free tier 2-node cap)
- Rate limiting
- OTel Collector config generation
- Backend switching (Grafana/Prometheus/Datadog)
- Bulk operations
- Web terminal proxy
- Multi-tenancy optimization

## Database

Uses SQLite with WAL mode. Schema is auto-created on startup. No manual migrations needed.

## Running with the Agent

1. Start the backend
2. Install the agent with a custom API URL:
   ```bash
   API_URL=http://localhost:8080 ./nodestral-agent
   ```
3. Agent registers and starts sending heartbeats
4. Use the dashboard to view your nodes

## License

MIT
