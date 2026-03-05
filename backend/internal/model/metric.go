package model

import "time"

// MetricSnapshot represents a single reading of system resource usage.
type MetricSnapshot struct {
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	MemoryUsagePercent float64   `json:"memory_usage_percent"`
	MemoryUsedBytes    uint64    `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64    `json:"memory_total_bytes"`
	DiskUsagePercent   float64   `json:"disk_usage_percent"`
	DiskUsedBytes      uint64    `json:"disk_used_bytes"`
	DiskTotalBytes     uint64    `json:"disk_total_bytes"`
	CollectedAt        time.Time `json:"collected_at"`
}

type SystemInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Architecture    string `json:"architecture"`
	CPUCores        int    `json:"cpu_cores"`
}
