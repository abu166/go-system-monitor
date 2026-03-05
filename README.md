# System Monitor (Go + React + Docker)

A production-style system monitoring app with:
- live CPU, memory, and disk monitoring
- threshold-based alerts
- persistent metric history
- Prometheus metrics and Grafana dashboards
- SSE live streaming to frontend

## Architecture Summary

### Backend (`backend`)
- `cmd/server`: startup, wiring, graceful shutdown
- `internal/config`: env config + validation
- `internal/collector`: metric/system collection via `gopsutil`
- `internal/service`: business logic, alert evaluation, persistence coordination
- `internal/storage`: in-memory history + JSONL persistent store
- `internal/api`: HTTP routes, middleware, error responses, SSE endpoint
- `internal/telemetry`: Prometheus instrumentation

### Frontend (`frontend`)
- React + TypeScript + Vite
- SSE live updates (`/api/metrics/stream`) with fallback polling
- alert rendering (critical/non-critical)
- loading/error/retry states

### Monitoring (`monitoring`)
- Prometheus scraping backend `/metrics`
- Grafana pre-provisioned datasource + dashboard

### Testing (`tests`, `.github/workflows`)
- backend unit test for alerting
- e2e smoke test script
- k6 load smoke test
- CI pipeline for build/test/smoke/load

## Folder Structure

```text
.
├── .env.example
├── .gitignore
├── docker-compose.yml
├── README.md
├── .github/workflows/ci.yml
├── backend
│   ├── Dockerfile
│   ├── go.mod
│   ├── cmd/server/main.go
│   └── internal
│       ├── api
│       ├── collector
│       ├── config
│       ├── model
│       ├── service
│       ├── storage
│       └── telemetry
├── frontend
│   ├── Dockerfile
│   ├── nginx.conf
│   └── src
├── monitoring
│   ├── prometheus/prometheus.yml
│   └── grafana/provisioning
└── tests
    ├── e2e/smoke.sh
    └── load/k6-smoke.js
```

## Run With Docker (Recommended)

1. Copy env template:

```bash
cp .env.example .env
```

2. Build and run all services:

```bash
docker compose up -d --build
```

3. Open:
- Frontend dashboard: `http://localhost:3000`
- Backend health: `http://localhost:8080/health`
- Backend Prometheus metrics: `http://localhost:8080/metrics`
- Prometheus UI: `http://localhost:9090`
- Grafana UI: `http://localhost:3001` (`admin` / `admin` by default)

4. Follow logs:

```bash
docker compose logs -f backend frontend prometheus grafana
```

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

### Optional local monitoring
You can still scrape `http://localhost:8080/metrics` from local Prometheus/Grafana instances.

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
- `VITE_POLL_INTERVAL_MS` (fallback interval, default `5000`)
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

## Alerting Behavior

Alerts trigger when current usage exceeds thresholds:
- CPU: `BACKEND_CPU_ALERT_THRESHOLD`
- Memory: `BACKEND_MEMORY_ALERT_THRESHOLD`
- Disk: `BACKEND_DISK_ALERT_THRESHOLD`

Frontend shows active alerts in a dedicated alert panel.

## Persistence Behavior

History is stored in `BACKEND_HISTORY_FILE` as JSONL.
On startup, backend loads past records and keeps the latest `BACKEND_HISTORY_LIMIT` in memory.

## Testing

### Backend unit tests

```bash
cd backend
go test ./...
```

### E2E smoke test (requires running backend)

```bash
./tests/e2e/smoke.sh
```

### k6 load smoke test (with Docker network)

```bash
docker run --rm \
  --network go-system-monitor_default \
  -e BASE_URL=http://backend:8080 \
  -v "$PWD/tests/load:/scripts" \
  grafana/k6:0.56.0 run /scripts/k6-smoke.js
```

## Troubleshooting

- Port conflict (`address already in use`):
  - update `.env` ports (`BACKEND_PORT`, `FRONTEND_PORT`, `PROMETHEUS_PORT`, `GRAFANA_PORT`)
- Frontend shows fallback/disconnected mode:
  - check `/api/metrics/stream` reachability
  - inspect `docker compose logs frontend backend`
- Grafana dashboard missing:
  - verify provisioning mounts in `docker-compose.yml`
  - check `docker compose logs grafana`
- History not persistent:
  - verify `backend_data` volume exists: `docker volume ls`
- High request volume in logs:
  - avoid multiple open tabs hitting dashboard
  - SSE should reduce request frequency vs short polling

## Completed Improvements

- Prometheus metrics export and Grafana dashboards
- WebSocket/SSE streaming instead of polling (implemented with SSE + fallback)
- persistent storage for historical metrics
- e2e and load testing pipeline
- alerting thresholds (CPU/memory/disk)
