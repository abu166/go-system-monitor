package collector

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type Collector interface {
	CollectMetrics(ctx context.Context) (model.MetricSnapshot, error)
	GetSystemInfo(ctx context.Context) (model.SystemInfo, error)
}

type SystemCollector struct {
	diskPath string
}

func NewSystemCollector(diskPath string) *SystemCollector {
	return &SystemCollector{diskPath: diskPath}
}

func (c *SystemCollector) CollectMetrics(ctx context.Context) (model.MetricSnapshot, error) {
	cpuUsage, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil {
		return model.MetricSnapshot{}, fmt.Errorf("read cpu usage: %w", err)
	}
	if len(cpuUsage) == 0 {
		return model.MetricSnapshot{}, fmt.Errorf("read cpu usage: empty result")
	}

	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return model.MetricSnapshot{}, fmt.Errorf("read memory usage: %w", err)
	}

	diskStat, err := disk.UsageWithContext(ctx, c.diskPath)
	if err != nil {
		return model.MetricSnapshot{}, fmt.Errorf("read disk usage for path %q: %w", c.diskPath, err)
	}

	return model.MetricSnapshot{
		CPUUsagePercent:    cpuUsage[0],
		MemoryUsagePercent: vmStat.UsedPercent,
		MemoryUsedBytes:    vmStat.Used,
		MemoryTotalBytes:   vmStat.Total,
		DiskUsagePercent:   diskStat.UsedPercent,
		DiskUsedBytes:      diskStat.Used,
		DiskTotalBytes:     diskStat.Total,
		CollectedAt:        time.Now().UTC(),
	}, nil
}

func (c *SystemCollector) GetSystemInfo(ctx context.Context) (model.SystemInfo, error) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return model.SystemInfo{}, fmt.Errorf("read host info: %w", err)
	}

	cpuCores, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		return model.SystemInfo{}, fmt.Errorf("read cpu cores: %w", err)
	}

	return model.SystemInfo{
		Hostname:        hostInfo.Hostname,
		OS:              hostInfo.OS,
		Platform:        hostInfo.Platform,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion:   hostInfo.KernelVersion,
		Architecture:    runtime.GOARCH,
		CPUCores:        cpuCores,
	}, nil
}
