package telemetry

import (
	"sync"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	CPUUsage      prometheus.Gauge
	MemoryUsage   prometheus.Gauge
	DiskUsage     prometheus.Gauge
	CollectErrors *prometheus.CounterVec
	APIRequests   *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec
}

var registerOnce sync.Once

func New() *Metrics {
	m := &Metrics{
		CPUUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "system_monitor_cpu_usage_percent",
			Help: "Current CPU usage percent",
		}),
		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "system_monitor_memory_usage_percent",
			Help: "Current memory usage percent",
		}),
		DiskUsage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "system_monitor_disk_usage_percent",
			Help: "Current disk usage percent",
		}),
		CollectErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "system_monitor_collection_errors_total",
			Help: "Total collector errors by operation",
		}, []string{"operation"}),
		APIRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "system_monitor_http_requests_total",
			Help: "Total HTTP requests by method/path/status",
		}, []string{"method", "path", "status"}),
		APIRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "system_monitor_http_request_duration_seconds",
			Help:    "HTTP request duration by method/path/status",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),
	}

	registerOnce.Do(func() {
		prometheus.MustRegister(m.CPUUsage, m.MemoryUsage, m.DiskUsage, m.CollectErrors, m.APIRequests, m.APIRequestDuration)
	})
	return m
}

func (m *Metrics) ObserveSnapshot(snapshot model.MetricSnapshot) {
	m.CPUUsage.Set(snapshot.CPUUsagePercent)
	m.MemoryUsage.Set(snapshot.MemoryUsagePercent)
	m.DiskUsage.Set(snapshot.DiskUsagePercent)
}
