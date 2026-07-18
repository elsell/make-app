import assert from 'node:assert/strict';
import test from 'node:test';
import { createApiClient } from './index.js';

test('authentication preserves headers serialized from the OpenAPI contract', async (context) => {
  const originalFetch = globalThis.fetch;
  let captured: Request | undefined;
  globalThis.fetch = async (input, init) => {
    captured = new Request(input, init);
    return new Response(JSON.stringify({ data: { id: 'example-1', name: 'Example' } }), {
      status: 201,
      headers: { 'Content-Type': 'application/json' },
    });
  };
  context.after(() => { globalThis.fetch = originalFetch; });

  const result = await createApiClient('https://api.example.test', () => 'application-session').POST('/v1/examples', {
    params: { header: { 'Idempotency-Key': '0123456789abcdef' } },
    body: { name: 'Example' },
  });

  assert.equal(result.response.status, 201);
  assert.equal(captured?.headers.get('authorization'), 'Bearer application-session');
  assert.equal(captured?.headers.get('idempotency-key'), '0123456789abcdef');
  assert.equal(captured?.headers.get('content-type'), 'application/json');
});
