import assert from 'node:assert/strict';
import test from 'node:test';
import { createApiClient } from './index.js';

test('authentication preserves headers serialized from the OpenAPI contract', async (context) => {
  const originalFetch = globalThis.fetch;
  let captured: Request | undefined;
  globalThis.fetch = async (input, init) => {
    captured = new Request(input, init);
    return new Response(JSON.stringify({ data: { id: 'user-1', email: 'person@example.test', displayName: 'Person' } }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    });
  };
  context.after(() => { globalThis.fetch = originalFetch; });

  const result = await createApiClient('https://api.example.test', () => 'application-session').GET('/v1/me');

  assert.equal(result.response.status, 200);
  assert.equal(captured?.headers.get('authorization'), 'Bearer application-session');
});
