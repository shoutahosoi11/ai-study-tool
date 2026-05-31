import { expect, request, test } from '@playwright/test';
import { e2eAPIBaseURL, shouldRunAPITests } from '../support/env';

test.describe('security API smoke', function () {
  test.skip(!shouldRunAPITests(), 'Set E2E_RUN_API_TESTS=true when a disposable backend is available');

  test('admin API rejects unauthenticated requests', async function () {
    const api = await request.newContext({ baseURL: e2eAPIBaseURL() });
    const response = await api.get('/api/v1/admin/overview');
    expect([401, 403]).toContain(response.status());
    await api.dispose();
  });

  test('extension import rejects missing extension token', async function () {
    const api = await request.newContext({ baseURL: e2eAPIBaseURL() });
    const response = await api.post('/api/v1/extension/highlights/import', {
      data: { highlights: [] },
    });
    expect([401, 403, 429]).toContain(response.status());
    await api.dispose();
  });

  test('security headers are present on health responses when backend is running', async function () {
    const api = await request.newContext({ baseURL: e2eAPIBaseURL() });
    const response = await api.get('/health');
    expect(response.status()).toBeLessThan(500);
    expect(response.headers()['x-content-type-options']).toBeTruthy();
    await api.dispose();
  });
});
