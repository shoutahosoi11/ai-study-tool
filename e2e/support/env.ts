const productionHostPatterns = [
  /\.run\.app$/i,
  /\.appspot\.com$/i,
  /\.web\.app$/i,
  /\.firebaseapp\.com$/i,
];

export function e2eBaseURL() {
  return normalizeURL(process.env.E2E_BASE_URL || 'http://127.0.0.1:3000');
}

export function e2eAPIBaseURL() {
  return normalizeURL(process.env.E2E_API_BASE_URL || 'http://127.0.0.1:8080');
}

export function e2eDryRun() {
  return process.env.E2E_DRY_RUN !== 'false';
}

export function shouldRunAPITests() {
  return process.env.E2E_RUN_API_TESTS === 'true';
}

export function assertSafeE2EEnvironment() {
  const allowProduction = process.env.E2E_ALLOW_PRODUCTION === 'true';
  for (const value of [e2eBaseURL(), e2eAPIBaseURL()]) {
    const parsed = new URL(value);
    const hostname = parsed.hostname.toLowerCase();
    const looksProduction =
      parsed.protocol === 'https:' &&
      hostname !== 'localhost' &&
      hostname !== '127.0.0.1' &&
      !hostname.endsWith('.local') &&
      productionHostPatterns.some(function (pattern) {
        return pattern.test(hostname);
      });

    if (looksProduction && !allowProduction) {
      throw new Error('Refusing to run E2E against a production-like URL without E2E_ALLOW_PRODUCTION=true');
    }
  }
}

function normalizeURL(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    throw new Error('E2E URL environment value must not be empty');
  }
  return trimmed.replace(/\/$/, '');
}
