import http from 'k6/http';
import { check, sleep } from 'k6';
import { requireBaseUrl, jsonHeaders } from './common.js';

export const options = {
  vus: 1,
  duration: '45s',
  thresholds: {
    http_req_failed: ['rate<0.30'],
    http_req_duration: ['p(95)<2000'],
  },
};

const baseUrl = requireBaseUrl();
const extensionToken = __ENV.EXTENSION_TOKEN || '';

if (!extensionToken) {
  throw new Error('EXTENSION_TOKEN is required');
}

export default function () {
  const payload = {
    highlights: [],
    source: 'loadtest',
  };
  const res = http.post(
    `${baseUrl}/api/v1/extension/highlights/import`,
    JSON.stringify(payload),
    { headers: jsonHeaders({ Authorization: `Bearer ${extensionToken}` }) },
  );

  check(res, {
    'import endpoint is controlled': (r) => [200, 202, 400, 401, 403, 429].includes(r.status),
  });
  sleep(2);
}

