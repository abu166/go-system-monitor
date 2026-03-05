export interface MetricSnapshot {
  cpu_usage_percent: number;
  memory_usage_percent: number;
  memory_used_bytes: number;
  memory_total_bytes: number;
  disk_usage_percent: number;
  disk_used_bytes: number;
  disk_total_bytes: number;
  collected_at: string;
}

export interface SystemInfo {
  hostname: string;
  os: string;
  platform: string;
  platform_version: string;
  kernel_version: string;
  architecture: string;
  cpu_cores: number;
}

export interface AlertItem {
  resource: string;
  value: number;
  threshold: number;
  message: string;
  is_critical: boolean;
}

export interface AlertStatus {
  triggered: boolean;
  alerts: AlertItem[];
  evaluated_at: string;
}

export interface MetricsStreamEvent {
  snapshot: MetricSnapshot;
  alerts: AlertStatus;
}

export interface ApiSuccess<T> {
  success: true;
  data: T;
}

export interface ApiFailure {
  success: false;
  error: string;
}

export type ApiResponse<T> = ApiSuccess<T> | ApiFailure;
