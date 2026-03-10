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
- Alertmanager UI: `http://localhost:9093` (when `monitoring` profile is enabled)
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
- `BACKEND_HISTORY_MAX_AGE` (default `24h`)
- `BACKEND_HISTORY_MAX_FILE_SIZE_BYTES` (default `10485760`)
- `BACKEND_HISTORY_IN_MEMORY_ONLY` (default `false`)
- `BACKEND_HISTORY_FALLBACK_TO_MEMORY` (default `true`)
- `BACKEND_READ_TIMEOUT` (default `5s`)
- `BACKEND_READ_HEADER_TIMEOUT` (default `2s`)
- `BACKEND_WRITE_TIMEOUT` (default `5s`)
- `BACKEND_IDLE_TIMEOUT` (default `30s`)
- `BACKEND_MAX_HEADER_BYTES` (default `1048576`)
- `BACKEND_SHUTDOWN_TIMEOUT` (default `10s`)
- `BACKEND_DISK_PATH` (default `/`)
- `BACKEND_STREAM_INTERVAL` (default `3s`)
- `BACKEND_HISTORY_FILE` (default `/app/data/metrics-history.jsonl`)
- `BACKEND_CORS_ALLOWED_ORIGINS` (default `http://localhost:3000,http://localhost`)
- `BACKEND_LOG_SAMPLE_RATE` (default `5`)
- `BACKEND_CPU_ALERT_THRESHOLD` (default `85`)
- `BACKEND_MEMORY_ALERT_THRESHOLD` (default `85`)
- `BACKEND_DISK_ALERT_THRESHOLD` (default `90`)

### Frontend
- `FRONTEND_PORT` (default `3000`)
- `VITE_API_BASE_URL` (default empty, same-origin)
- `VITE_POLL_INTERVAL_MS` (default `5000`)
- `VITE_REQUEST_TIMEOUT_MS` (default `4000`)
- `VITE_STREAM_RECONNECT_BASE_MS` (default `1000`)
- `VITE_STREAM_RECONNECT_MAX_MS` (default `30000`)

### Monitoring
- `PROMETHEUS_PORT` (default `9090`)
- `ALERTMANAGER_PORT` (default `9093`)
- `GRAFANA_PORT` (default `3001`)
- `GRAFANA_ADMIN_USER` (default `admin`)
- `GRAFANA_ADMIN_PASSWORD` (default `admin`)

## API Endpoints

- `GET /health`
- `GET /live`
- `GET /ready`
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

## Prometheus Alerts

Alerting rules are configured in `monitoring/prometheus/alerts.yml` and loaded by Prometheus.
Prometheus forwards alerts to Alertmanager using `monitoring/alertmanager/alertmanager.yml`.

- Open Prometheus Alerts page: `http://localhost:9090/alerts`
- Open Alertmanager page: `http://localhost:9093`
- You should see rules like:
  - `SystemMonitorBackendDown`
  - `SystemMonitorHighCPUUsage`
  - `SystemMonitorHighMemoryUsage`
  - `SystemMonitorHighDiskUsage`
  - `SystemMonitorCollectionErrorRate`
  - `SystemMonitorHighHTTP5xxRatio`
  - `SystemMonitorHighP95Latency`
  - `SystemMonitorLowAvailability`

Grafana dashboard also includes an `Active Alerts (firing)` chart that reads from Prometheus `ALERTS` series.

Additional SLO/operability panels:
- API p95 latency
- availability ratio
- error budget burn (99.9% SLO)
- alert history (24h)
- top failing endpoints

## Test Notes

- Go integration/unit tests are under `backend/internal/...`.
- Frontend stream/backoff unit tests use `vitest`.
- Prometheus rules tests use `promtool test rules` with `monitoring/prometheus/alerts.test.yml`.

@abu166
