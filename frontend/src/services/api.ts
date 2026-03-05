import { AlertStatus, ApiResponse, MetricSnapshot, MetricsStreamEvent, SystemInfo } from '../types/metrics';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';
const REQUEST_TIMEOUT_MS = Number(import.meta.env.VITE_REQUEST_TIMEOUT_MS ?? 4000);

class ApiClientError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ApiClientError';
  }
}

function withTimeout(signal: AbortSignal | undefined): AbortSignal {
  if (signal?.aborted) return signal;

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  signal?.addEventListener('abort', () => controller.abort(), { once: true });
  controller.signal.addEventListener('abort', () => clearTimeout(timeout), { once: true });

  return controller.signal;
}

async function getJson<T>(path: string, signal?: AbortSignal): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      signal: withTimeout(signal)
    });
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new ApiClientError(`request timeout after ${REQUEST_TIMEOUT_MS}ms`);
    }
    throw new ApiClientError('network error while calling API');
  }

  let json: ApiResponse<T>;
  try {
    json = (await response.json()) as ApiResponse<T>;
  } catch {
    throw new ApiClientError('malformed JSON response from server');
  }

  if (!response.ok) {
    if (!json || typeof json !== 'object' || !('error' in json)) {
      throw new ApiClientError(`request failed with status ${response.status}`);
    }
    throw new ApiClientError(String(json.error));
  }

  if (!json || typeof json !== 'object' || !('success' in json)) {
    throw new ApiClientError('unexpected response shape');
  }

  if (!json.success) {
    throw new ApiClientError(json.error || 'API returned unsuccessful response');
  }

  return json.data;
}

export async function fetchLatestMetrics(signal?: AbortSignal): Promise<MetricSnapshot> {
  return getJson<MetricSnapshot>('/api/metrics/latest', signal);
}

export async function fetchSystemInfo(signal?: AbortSignal): Promise<SystemInfo> {
  return getJson<SystemInfo>('/api/system/info', signal);
}

export async function fetchCurrentAlerts(signal?: AbortSignal): Promise<AlertStatus> {
  return getJson<AlertStatus>('/api/alerts/current', signal);
}

export function openMetricsStream(
  onData: (event: MetricsStreamEvent) => void,
  onError: (message: string) => void
): EventSource {
  const stream = new EventSource(`${API_BASE_URL}/api/metrics/stream`);

  stream.addEventListener('metrics', (event) => {
    try {
      const parsed = JSON.parse((event as MessageEvent).data) as MetricsStreamEvent;
      if (!parsed?.snapshot || !parsed?.alerts) {
        throw new Error('missing fields');
      }
      onData(parsed);
    } catch {
      onError('failed to parse live stream data');
    }
  });

  stream.onerror = () => {
    onError('live stream disconnected, using fallback refresh');
  };

  return stream;
}

export { ApiClientError };
