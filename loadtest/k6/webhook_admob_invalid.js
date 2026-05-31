import http from 'k6/http';
import { check, sleep } from 'k6';
import { requireBaseUrl } from './common.js';

export const options = {
  vus: 1,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.50'],
    http_req_duration: ['p(95)<1500'],
  },
};

const baseUrl = requireBaseUrl();

export default function () {
  const query = [
    'ad_unit=test-ad-unit',
    'reward_amount=1',
    'reward_item=token',
    'transaction_id=loadtest-invalid',
    'user_id=00000000-0000-0000-0000-000000000000',
    'timestamp=1',
    'signature=invalid',
    'key_id=invalid',
  ].join('&');

  const res = http.get(`${baseUrl}/webhooks/admob/ssv?${query}`);
  check(res, {
    'invalid ssv is rejected or rate limited': (r) => [400, 401, 403, 429].includes(r.status),
  });
  sleep(2);
}

