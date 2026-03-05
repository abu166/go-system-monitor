import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import MetricCard from './components/MetricCard';
import StatusBanner from './components/StatusBanner';
import { fetchCurrentAlerts, fetchLatestMetrics, fetchSystemInfo, openMetricsStream } from './services/api';
import { AlertStatus, MetricSnapshot, SystemInfo } from './types/metrics';

const FALLBACK_POLL_MS = Number(import.meta.env.VITE_POLL_INTERVAL_MS ?? 5000);

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return 'N/A';
  if (bytes === 0) return '0 B';

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const exp = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / 1024 ** exp;
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[exp]}`;
}

function formatPercent(value: number): string {
  if (!Number.isFinite(value) || value < 0) return 'N/A';
  return `${value.toFixed(1)}%`;
}

export default function App() {
  const [metrics, setMetrics] = useState<MetricSnapshot | null>(null);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [alerts, setAlerts] = useState<AlertStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<string>('Never');
  const [live, setLive] = useState(false);
  const abortRef = useRef<AbortController | null>(null);
  const streamRef = useRef<EventSource | null>(null);

  const loadFallbackData = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    try {
      setError(null);
      const [metricsData, infoData, alertData] = await Promise.all([
        fetchLatestMetrics(controller.signal),
        fetchSystemInfo(controller.signal),
        fetchCurrentAlerts(controller.signal)
      ]);

      setMetrics(metricsData);
      setSystemInfo(infoData);
      setAlerts(alertData);
      setLastUpdated(new Date(metricsData.collected_at).toLocaleString());
    } catch (err) {
      const message = err instanceof Error ? err.message : 'unknown error while loading data';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadFallbackData();

    streamRef.current?.close();
    const stream = openMetricsStream(
      (event) => {
        setLive(true);
        setError(null);
        setMetrics(event.snapshot);
        setAlerts(event.alerts);
        setLastUpdated(new Date(event.snapshot.collected_at).toLocaleString());
        setLoading(false);
      },
      (message) => {
        setLive(false);
        setError(message);
      }
    );
    streamRef.current = stream;

    const timer = window.setInterval(() => {
      if (!live) {
        void loadFallbackData();
      }
    }, FALLBACK_POLL_MS);

    return () => {
      window.clearInterval(timer);
      abortRef.current?.abort();
      stream.close();
    };
  }, [loadFallbackData, live]);

  useEffect(() => {
    if (!systemInfo) {
      const controller = new AbortController();
      void fetchSystemInfo(controller.signal)
        .then((info) => setSystemInfo(info))
        .catch(() => undefined);
      return () => controller.abort();
    }
    return undefined;
  }, [systemInfo]);

  const metricCards = useMemo(
    () => [
      {
        title: 'CPU Usage',
        value: metrics ? formatPercent(metrics.cpu_usage_percent) : 'N/A',
        subtitle: 'Current utilization'
      },
      {
        title: 'Memory Usage',
        value: metrics ? formatPercent(metrics.memory_usage_percent) : 'N/A',
        subtitle: metrics
          ? `${formatBytes(metrics.memory_used_bytes)} / ${formatBytes(metrics.memory_total_bytes)}`
          : undefined
      },
      {
        title: 'Disk Usage',
        value: metrics ? formatPercent(metrics.disk_usage_percent) : 'N/A',
        subtitle: metrics
          ? `${formatBytes(metrics.disk_used_bytes)} / ${formatBytes(metrics.disk_total_bytes)}`
          : undefined
      }
    ],
    [metrics]
  );

  return (
    <main className="page">
      <header>
        <h1>System Monitor</h1>
        <p className="subtitle">Live host metrics dashboard ({live ? 'SSE live' : 'fallback mode'})</p>
      </header>

      <StatusBanner loading={loading} error={error} onRetry={loadFallbackData} />

      {alerts?.triggered ? (
        <section className="alert-card" aria-live="assertive">
          <h2>Active Alerts</h2>
          <ul>
            {alerts.alerts.map((alert) => (
              <li key={`${alert.resource}-${alert.threshold}`} className={alert.is_critical ? 'critical' : ''}>
                {alert.message}: {alert.value.toFixed(1)}% (threshold {alert.threshold.toFixed(1)}%)
              </li>
            ))}
          </ul>
        </section>
      ) : (
        <section className="alert-card ok">No active threshold alerts.</section>
      )}

      <section className="grid" aria-live="polite">
        {metricCards.map((card) => (
          <MetricCard key={card.title} title={card.title} value={card.value} subtitle={card.subtitle} />
        ))}
      </section>

      <section className="meta-card">
        <h2>Host Information</h2>
        {systemInfo ? (
          <ul>
            <li>Hostname: {systemInfo.hostname}</li>
            <li>OS: {systemInfo.os}</li>
            <li>
              Platform: {systemInfo.platform} {systemInfo.platform_version}
            </li>
            <li>Kernel: {systemInfo.kernel_version}</li>
            <li>Architecture: {systemInfo.architecture}</li>
            <li>CPU Cores: {systemInfo.cpu_cores}</li>
          </ul>
        ) : (
          <p>No host information available.</p>
        )}
      </section>

      <footer>Last updated: {lastUpdated}</footer>
    </main>
  );
}
