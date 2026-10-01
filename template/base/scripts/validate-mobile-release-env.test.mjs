import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const script = fileURLToPath(new URL('./validate-mobile-release-env.mjs', import.meta.url));
const valid = {
  __ENV_PREFIX___APP_ENV: 'production',
  __ENV_PREFIX___API_URL: 'https://api.example.com',
  __ENV_PREFIX___OIDC_ISSUER: 'https://identity.example.com',
  __ENV_PREFIX___MOBILE_OIDC_CLIENT_ID: '__APP_SLUG__-mobile',
  __ENV_PREFIX___EXPO_PROJECT_ID: '3f14cb3e-753d-48eb-89e4-54df740de5c4',
};

function validate(overrides = {}) {
  execFileSync(process.execPath, [script], { env: { ...valid, ...overrides }, stdio: 'pipe' });
}

test('signed release validation accepts explicit public production endpoints', () => {
  assert.doesNotThrow(() => validate());
});

test('signed release validation rejects unspecified and IPv4-mapped local IPv6 endpoints', () => {
  for (const endpoint of ['https://[::]', 'https://[::ffff:7f00:1]', 'https://[::ffff:a00:204]', 'https://[::ffff:ac14:102]', 'https://[::ffff:c0a8:102]', 'https://[::ffff:a9fe:304]']) {
    assert.throws(() => validate({ __ENV_PREFIX___API_URL: endpoint }), endpoint);
    assert.throws(() => validate({ __ENV_PREFIX___OIDC_ISSUER: endpoint }), endpoint);
  }
});
