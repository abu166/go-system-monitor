import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500']
  }
};

const BASE_URL = __ENV.BASE_URL || 'http://backend:8080';

export default function () {
  const metricsRes = http.get(`${BASE_URL}/api/metrics/latest`);
  check(metricsRes, {
    'metrics status is 200': (r) => r.status === 200,
    'metrics has success=true': (r) => r.body.includes('"success":true')
  });

  const infoRes = http.get(`${BASE_URL}/api/system/info`);
  check(infoRes, {
    'info status is 200': (r) => r.status === 200,
    'info has hostname': (r) => r.body.includes('"hostname"')
  });

  sleep(1);
}
