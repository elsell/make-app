import assert from 'node:assert/strict';
import test from 'node:test';
import { createApiClient } from './index.js';

async function capturedRequest(token: string | null) {
  const originalFetch = globalThis.fetch;
  let captured: Request | undefined;
  globalThis.fetch = async (input, init) => {
    captured = new Request(input, init);
    return new Response(JSON.stringify({ data: { id: 'example-1', name: 'Example' } }), {
      status: 201,
      headers: { 'Content-Type': 'application/json' },
    });
  };
  try {
    const result = await createApiClient('https://api.example.test', () => token).POST('/v1/examples', {
      params: { header: { Authorization: 'Bearer stale-session', 'Idempotency-Key': '0123456789abcdef' } },
      headers: { 'X-Caller-Header': 'caller-value' },
      body: { name: 'Example' },
    });
    assert.equal(result.response.status, 201);
    assert.ok(captured);
    return captured;
  } finally {
    globalThis.fetch = originalFetch;
  }
}

test('a current token owns authorization while preserving every unrelated header', async () => {
  const captured = await capturedRequest('application-session');
  assert.equal(captured.headers.get('authorization'), 'Bearer application-session');
  assert.equal(captured.headers.get('idempotency-key'), '0123456789abcdef');
  assert.equal(captured.headers.get('content-type'), 'application/json');
  assert.equal(captured.headers.get('x-caller-header'), 'caller-value');
});

for (const [name, token] of [['null', null], ['empty', '']] as const) {
  test(`${name} token removes caller-supplied stale authorization without dropping other headers`, async () => {
    const captured = await capturedRequest(token);
    assert.equal(captured.headers.get('authorization'), null);
    assert.equal(captured.headers.get('idempotency-key'), '0123456789abcdef');
    assert.equal(captured.headers.get('content-type'), 'application/json');
    assert.equal(captured.headers.get('x-caller-header'), 'caller-value');
  });
}
