import http from 'k6/http';
import { check, sleep } from 'k6';
import { requireBaseUrl, jsonHeaders } from './common.js';

export const options = {
  vus: 1,
  iterations: 5,
  thresholds: {
    http_req_failed: ['rate<0.30'],
    http_req_duration: ['p(95)<2500'],
  },
};

const enabled = (__ENV.QUESTION_GENERATION_LOADTEST_ENABLED || '').toLowerCase() === 'true';
if (!enabled) {
  throw new Error('Refusing to run without QUESTION_GENERATION_LOADTEST_ENABLED=true');
}

const baseUrl = requireBaseUrl();
const authToken = __ENV.AUTH_TOKEN || '';

if (!authToken) {
  throw new Error('AUTH_TOKEN is required');
}

export default function () {
  const res = http.post(
    `${baseUrl}/api/v1/questions/sync`,
    JSON.stringify({ reason: 'loadtest' }),
    { headers: jsonHeaders({ Authorization: `Bearer ${authToken}` }) },
  );

  check(res, {
    'sync endpoint is controlled': (r) => [200, 202, 400, 401, 403, 429].includes(r.status),
  });
  sleep(3);
}

