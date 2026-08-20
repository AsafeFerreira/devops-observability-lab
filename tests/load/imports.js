import http from 'k6/http';
import { check } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';
const duration = __ENV.LAB_K6_DURATION || '30s';
const rate = Number(__ENV.LAB_K6_RATE || 10);

export const options = {
  scenarios: {
    imports: {
      executor: 'constant-arrival-rate',
      rate,
      timeUnit: '1s',
      duration,
      preAllocatedVUs: Math.max(rate, 10),
      maxVUs: Math.max(rate * 3, 30),
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<750'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const unique = `${__VU}-${__ITER}-${Date.now()}`;
  const response = http.post(
    `${baseURL}/api/v1/imports`,
    JSON.stringify({ source: 'load-test', recordCount: 10 }),
    {
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': `client-${(__VU % 3) + 1}`,
        'X-Correlation-ID': `k6-${unique}`,
        'Idempotency-Key': `k6-${unique}`,
      },
      tags: { name: 'POST /api/v1/imports' },
    },
  );

  check(response, {
    'import accepted': (result) => result.status === 201,
    'response has import id': (result) => Boolean(result.json('id')),
  });
}

export function handleSummary(data) {
  const rateMetric = data.metrics.http_req_failed?.values.rate ?? 0;
  const p95 = data.metrics.http_req_duration?.values['p(95)'] ?? 0;
  return {
    stdout: `\nLoad test: failed=${(rateMetric * 100).toFixed(2)}% p95=${p95.toFixed(1)}ms\n`,
    '/artifacts/k6-summary.json': JSON.stringify(data, null, 2),
  };
}
