// Optional read-only smoke check against the normal environment: ten requests.
import http from 'k6/http';
import { check, sleep } from 'k6';

if (!__ENV.BUOY_NAME) throw new Error('Set BUOY_NAME to an existing buoy');
const origin = (__ENV.API_ORIGIN || 'https://treblesurf.com').replace(/\/$/, '');
export const options = {
  scenarios: { smoke: { executor: 'per-vu-iterations', vus: 1, iterations: 10, maxDuration: '2m' } },
  thresholds: { checks: ['rate>0.99'] },
};
export default function () {
  const response = http.get(`${origin}/api/getLast24BuoyData?buoyName=${encodeURIComponent(__ENV.BUOY_NAME)}`, {
    timeout: '10s', tags: { name: '/api/getLast24BuoyData' },
  });
  check(response, { 'history responds successfully': (r) => r.status === 200 });
  sleep(1);
}
