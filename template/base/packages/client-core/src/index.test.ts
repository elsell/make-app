import assert from 'node:assert/strict';
import test from 'node:test';
import { classifySessionFailure, publicEndpointConfig, publicEnvironmentConfig, publicStringConfig, refreshSessionCredential, retainedSessionExpiry, sessionRetryDelay, validateSessionCredential, type SessionFailure } from './index.js';

test('network, 429, and 503 preserve a valid credential as authenticated_offline', () => {
  for (const failure of [{ kind: 'network' } as const, { kind: 'http', status: 429 } as const, { kind: 'http', status: 503 } as const]) {
    assert.deepEqual(classifySessionFailure(failure), {
      state: 'authenticated_offline', discardCredential: false, retryable: true,
    });
  }
});

test('401 and expired credentials require authentication and are discarded', () => {
  for (const failure of [{ kind: 'http', status: 401 } as const, { kind: 'expired' } as const]) {
    assert.deepEqual(classifySessionFailure(failure), {
      state: 'authentication_required', discardCredential: true, retryable: false,
    });
  }
});

test('non-401 HTTP failures preserve credentials and only transient statuses retry', () => {
  for (const status of [400, 403, 404]) {
    assert.deepEqual(classifySessionFailure({ kind: 'http', status }), {
      state: 'authenticated_offline', discardCredential: false, retryable: false,
    });
  }
  assert.deepEqual(classifySessionFailure({ kind: 'http', status: 408 }), {
    state: 'authenticated_offline', discardCredential: false, retryable: true,
  });
});

test('malformed local storage is classified separately and discarded', () => {
  assert.deepEqual(classifySessionFailure({ kind: 'local_storage', reason: 'malformed' }), {
    state: 'local_session_unreadable', discardCredential: true, retryable: false,
  });
});

const current = { token: 'old-token', expiresAt: '2026-07-18T12:00:00Z' };
const replacement = { token: 'new-token', expiresAt: '2026-07-18T13:00:00Z' };

test('refresh adapters classify network, 401, 429, and 503 without overwriting the credential', async () => {
  for (const [name, request, expected] of [
    ['network', async () => { throw new Error('offline'); }, { kind: 'network' }],
    ['401', async () => ({ ok: false, status: 401, json: async () => ({}) }), { kind: 'http', status: 401 }],
    ['429', async () => ({ ok: false, status: 429, json: async () => ({}) }), { kind: 'http', status: 429 }],
    ['503', async () => ({ ok: false, status: 503, json: async () => ({}) }), { kind: 'http', status: 503 }],
  ] as const) {
    const saved: unknown[] = [];
    await assert.rejects(
      refreshSessionCredential(current, request, async (value) => { saved.push(value); }),
      (failure: SessionFailure) => { assert.deepEqual(failure, expected, name); return true; },
    );
    assert.deepEqual(saved, [], `${name} must not replace storage`);
  }
});

test('refresh adapters atomically persist a validated replacement credential', async () => {
  const saved: unknown[] = [];
  const result = await refreshSessionCredential(
    current,
    async () => ({ ok: true, status: 200, json: async () => ({ data: replacement }) }),
    async (value) => { saved.push(value); },
  );
  assert.deepEqual(result, replacement);
  assert.deepEqual(saved, [replacement]);
});

test('a profile outage after rotation leaves the replacement credential persisted for recovery', async () => {
  let saved = current;
  const next = await refreshSessionCredential(
    current,
    async () => ({ ok: true, status: 200, json: async () => ({ data: replacement }) }),
    async (value) => { saved = value; },
  );
  await assert.rejects(
    validateSessionCredential(next, async () => { throw new Error('offline'); }),
    (failure: SessionFailure) => failure.kind === 'network',
  );
  assert.deepEqual(saved, replacement);
});

test('profile-outage retry timing follows the retained rotated credential', () => {
  const oldExpiry = '2026-07-18T12:05:00Z';
  const replacement = { token: 'replacement', expiresAt: '2026-07-18T13:00:00Z' };
  assert.equal(retainedSessionExpiry(replacement, oldExpiry), replacement.expiresAt);
  assert.equal(retainedSessionExpiry(null, oldExpiry), oldExpiry);
});

test('retry backoff is bounded by five minutes and credential expiry', () => {
  const now = Date.parse('2026-07-18T12:00:00Z');
  assert.equal(sessionRetryDelay(0, '2026-07-18T13:00:00Z', now), 30_000);
  assert.equal(sessionRetryDelay(20, '2026-07-18T13:00:00Z', now), 300_000);
  assert.equal(sessionRetryDelay(20, '2026-07-18T12:01:00Z', now), 60_000);
  assert.equal(sessionRetryDelay(0, '2026-07-18T11:59:00Z', now), 0);
});

test('production endpoint configuration fails closed for unsafe actual bundle values', () => {
  for (const value of [undefined, 'http://api.example.com', 'https://LOCALHOST/path', 'https://sub.localhost/path', 'https://127.0.0.2', 'https://[::1]', 'https://user:pass@example.com', 'https://api.example.com?target=dev', 'https://api.example.com/#dev']) {
    assert.throws(() => publicEndpointConfig(value, 'http://localhost:8080', 'EXPO_PUBLIC_API_URL', true));
  }
  assert.equal(publicEndpointConfig('https://api.example.com/', 'http://localhost:8080', 'API', true), 'https://api.example.com');
  assert.equal(publicEndpointConfig(undefined, 'http://localhost:8080', 'API', false), 'http://localhost:8080');
});

test('unknown deployment environment fails closed', () => {
  assert.equal(publicEnvironmentConfig(undefined, 'PUBLIC_APP_ENV'), 'development');
  assert.equal(publicEnvironmentConfig('production', 'PUBLIC_APP_ENV'), 'production');
  assert.throws(() => publicEnvironmentConfig('prod', 'PUBLIC_APP_ENV'));
});

test('missing production string configuration fails closed', () => {
  assert.equal(publicStringConfig(undefined, 'local-client', 'PUBLIC_OIDC_CLIENT_ID', false), 'local-client');
  assert.equal(publicStringConfig('production-client', 'local-client', 'PUBLIC_OIDC_CLIENT_ID', true), 'production-client');
  assert.throws(() => publicStringConfig(undefined, 'local-client', 'PUBLIC_OIDC_CLIENT_ID', true));
  assert.throws(() => publicStringConfig('   ', 'local-client', 'PUBLIC_OIDC_CLIENT_ID', true));
});
