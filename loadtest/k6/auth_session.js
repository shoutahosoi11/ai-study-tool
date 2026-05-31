import http from 'k6/http';
import { check, sleep } from 'k6';
import { requireBaseUrl, jsonHeaders } from './common.js';

export const options = {
  vus: 1,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.20'],
    http_req_duration: ['p(95)<1500'],
  },
};

const baseUrl = requireBaseUrl();
const idToken = __ENV.ID_TOKEN || '';

if (!idToken) {
  throw new Error('ID_TOKEN is required');
}

export default function () {
  const res = http.post(
    `${baseUrl}/api/v1/auth/session`,
    JSON.stringify({ id_token: idToken }),
    { headers: jsonHeaders() },
  );

  check(res, {
    'session endpoint returns expected class': (r) => [200, 400, 401, 429].includes(r.status),
  });
  sleep(2);
}

