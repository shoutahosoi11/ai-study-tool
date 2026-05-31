export function requireBaseUrl() {
  const baseUrl = (__ENV.BASE_URL || '').replace(/\/+$/, '');
  if (!baseUrl) {
    throw new Error('BASE_URL is required');
  }
  assertNotProduction(baseUrl);
  return baseUrl;
}

export function assertNotProduction(baseUrl) {
  const allowProduction = (__ENV.ALLOW_PRODUCTION_LOADTEST || '').toLowerCase() === 'true';
  const normalized = baseUrl.toLowerCase();
  const looksProduction =
    normalized.includes('ai-study-tool.com') ||
    normalized.includes('api.ai-study-tool') ||
    normalized.includes('run.app');

  if (looksProduction && !allowProduction) {
    throw new Error('Refusing production-looking BASE_URL without ALLOW_PRODUCTION_LOADTEST=true');
  }
}

export function jsonHeaders(extra = {}) {
  return {
    'Content-Type': 'application/json',
    ...extra,
  };
}

