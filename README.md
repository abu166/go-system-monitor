# System Monitor (Go + React + Docker)

A system monitoring app with:
- live CPU, memory, and disk monitoring
- threshold-based alerts
- persistent metric history
- Prometheus metrics and Grafana dashboards
- SSE live streaming to frontend

## Folder Structure

```text
.
├── .env.example
├── .gitignore
├── docker-compose.yml
├── README.md
├── backend
├── frontend
├── monitoring
└── tests
```

## Run With Docker

1. Copy env template:

```bash
cp .env.example .env
```

2. Fast default startup (app only: backend + frontend):

```bash
docker compose up -d --build
```

3. Full stack (app + Prometheus + Grafana):

```bash
docker compose --profile monitoring up -d --build
```

4. Open:
- Frontend dashboard: `http://localhost:3000`
- Backend health: `http://localhost:8080/health`
- Backend Prometheus metrics: `http://localhost:8080/metrics`
- Prometheus UI: `http://localhost:9090` (when `monitoring` profile is enabled)
- Grafana UI: `http://localhost:3001` (when `monitoring` profile is enabled)

## Docker Performance Tips

- Enable BuildKit/Bake for faster builds:

```bash
export COMPOSE_BAKE=true
```

- Rebuild only one changed service:

```bash
docker compose up -d --build backend
```

- Use app-only mode for daily development (`monitoring` profile disabled).
- Build caches are enabled in Dockerfiles for Go modules, Go build cache, and npm cache.
- `.dockerignore` files reduce build context transfer for backend/frontend.

## Stop / Shutdown

- Stop app-only stack (backend + frontend):

```bash
docker compose down
```

- Stop full stack including monitoring profile and remove volumes:

```bash
docker compose --profile monitoring down -v --remove-orphans
```

- If Docker says network is still in use, check attached containers:

```bash
docker ps --filter network=go-system-monitor_default
```

Then stop/remove those containers and run `down` again.

## Run Without Docker

### Backend

```bash
cd backend
go mod tidy
go run ./cmd/server
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

### Backend
- `BACKEND_PORT` (default `8080`)
- `BACKEND_HISTORY_LIMIT` (default `100`)
- `BACKEND_READ_TIMEOUT` (default `5s`)
- `BACKEND_WRITE_TIMEOUT` (default `5s`)
- `BACKEND_SHUTDOWN_TIMEOUT` (default `10s`)
- `BACKEND_DISK_PATH` (default `/`)
- `BACKEND_STREAM_INTERVAL` (default `3s`)
- `BACKEND_HISTORY_FILE` (default `/app/data/metrics-history.jsonl`)
- `BACKEND_CPU_ALERT_THRESHOLD` (default `85`)
- `BACKEND_MEMORY_ALERT_THRESHOLD` (default `85`)
- `BACKEND_DISK_ALERT_THRESHOLD` (default `90`)

### Frontend
- `FRONTEND_PORT` (default `3000`)
- `VITE_API_BASE_URL` (default empty, same-origin)
- `VITE_POLL_INTERVAL_MS` (default `5000`)
- `VITE_REQUEST_TIMEOUT_MS` (default `4000`)

### Monitoring
- `PROMETHEUS_PORT` (default `9090`)
- `GRAFANA_PORT` (default `3001`)
- `GRAFANA_ADMIN_USER` (default `admin`)
- `GRAFANA_ADMIN_PASSWORD` (default `admin`)

## API Endpoints

- `GET /health`
- `GET /metrics` (Prometheus)
- `GET /api/metrics/latest`
- `GET /api/metrics/history`
- `GET /api/metrics/stream` (SSE)
- `GET /api/system/info`
- `GET /api/alerts/current`

### Success format

```json
{
  "success": true,
  "data": {}
}
```

### Error format

```json
{
  "success": false,
  "error": "message"
}
```

## Postman

Import these files into Postman:
- `tests/postman/system-monitor.postman_collection.json`
- `tests/postman/system-monitor.postman_environment.json`

Then run the collection against `base_url=http://localhost:8080`.

## Troubleshooting

- Port conflict (`address already in use`):
  - update `.env` ports (`BACKEND_PORT`, `FRONTEND_PORT`, `PROMETHEUS_PORT`, `GRAFANA_PORT`)
- SSE errors on dashboard:
  - check backend logs and ensure `/api/metrics/stream` returns `200`
- Grafana dashboard missing:
  - verify provisioning mounts in `docker-compose.yml`
  - check `docker compose logs grafana`
