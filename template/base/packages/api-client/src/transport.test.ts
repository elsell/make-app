import assert from 'node:assert/strict';
import test from 'node:test';
import { createAuthenticatedFetch } from './transport.js';

async function capturedMergedRequest(token: string | null) {
  const originalFetch = globalThis.fetch;
  let captured: Request | undefined;
  globalThis.fetch = async (input, init) => {
    captured = new Request(input, init);
    return new Response(null, { status: 204 });
  };
  try {
    const input = new Request('https://api.example.test/v1/me', {
      headers: { Authorization: 'Bearer stale-input', 'X-Input-Header': 'input-value' },
    });
    await createAuthenticatedFetch(() => token)(input, {
      headers: { Authorization: 'Bearer stale-init', 'X-Init-Header': 'init-value' },
    });
    assert.ok(captured);
    return captured;
  } finally {
    globalThis.fetch = originalFetch;
  }
}

test('authenticated fetch merges Request and init headers before applying the current token', async () => {
  const captured = await capturedMergedRequest('application-session');
  assert.equal(captured.headers.get('authorization'), 'Bearer application-session');
  assert.equal(captured.headers.get('x-input-header'), 'input-value');
  assert.equal(captured.headers.get('x-init-header'), 'init-value');
});

for (const [name, token] of [['null', null], ['empty', '']] as const) {
  test(`authenticated fetch removes stale authorization when the token is ${name}`, async () => {
    const captured = await capturedMergedRequest(token);
    assert.equal(captured.headers.get('authorization'), null);
    assert.equal(captured.headers.get('x-input-header'), 'input-value');
    assert.equal(captured.headers.get('x-init-header'), 'init-value');
  });
}
