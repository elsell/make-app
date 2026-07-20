import assert from 'node:assert/strict';
import { createServer, type IncomingMessage } from 'node:http';
import test from 'node:test';
import { createSessionApiClient } from './index.js';

type CapturedRequest = { method: string; url: string; authorization: string | undefined; body: string };

async function bodyOf(request: IncomingMessage): Promise<string> {
  const chunks: Buffer[] = [];
  for await (const chunk of request) chunks.push(Buffer.from(chunk));
  return Buffer.concat(chunks).toString('utf8');
}

test('session API operations use the real generated transport and credential adapter', async () => {
  const requests: CapturedRequest[] = [];
  let profileStatus = 200;
  const server = createServer(async (request, response) => {
    requests.push({
      method: request.method ?? '',
      url: request.url ?? '',
      authorization: request.headers.authorization,
      body: await bodyOf(request),
    });
    response.setHeader('Content-Type', 'application/json');
    if (request.url === '/v1/sessions') {
      response.end(JSON.stringify({ data: { token: 'application-session', expiresAt: '2026-07-21T12:00:00Z' } }));
    } else if (request.url === '/v1/session/refresh') {
      response.end(JSON.stringify({ data: { token: 'replacement-session', expiresAt: '2026-07-22T12:00:00Z' } }));
    } else if (request.url === '/v1/me') {
      response.statusCode = profileStatus;
      response.end(profileStatus === 200
        ? JSON.stringify({ data: { id: 'user-1', email: 'user@example.test', displayName: 'User' } })
        : JSON.stringify({ status: 401, code: 'authentication_required', title: 'Unauthorized' }));
    } else if (request.url === '/v1/session') {
      response.statusCode = 204;
      response.removeHeader('Content-Type');
      response.end();
    } else {
      response.statusCode = 404;
      response.end(JSON.stringify({ status: 404, code: 'not_found', title: 'Not Found' }));
    }
  });
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const address = server.address();
    assert.ok(address && typeof address !== 'string');
    const baseUrl = `http://127.0.0.1:${address.port}`;

    const exchange = await createSessionApiClient(baseUrl, () => null).exchange('provider-token');
    assert.equal(exchange.response.status, 200);
    assert.equal(exchange.data?.data.token, 'application-session');

    const authenticated = createSessionApiClient(baseUrl, () => 'application-session');
    const refresh = await authenticated.refresh();
    assert.equal(refresh.data?.data.token, 'replacement-session');
    const profile = await authenticated.profile();
    assert.equal(profile.data?.data.id, 'user-1');
    const revoke = await authenticated.revoke();
    assert.equal(revoke.response.status, 204);

    profileStatus = 401;
    const rejected = await authenticated.profile();
    assert.equal(rejected.response.status, 401);
    assert.equal(rejected.data, undefined);

    assert.deepEqual(requests.map(({ method, url, authorization }) => ({ method, url, authorization })), [
      { method: 'POST', url: '/v1/sessions', authorization: undefined },
      { method: 'POST', url: '/v1/session/refresh', authorization: 'Bearer application-session' },
      { method: 'GET', url: '/v1/me', authorization: 'Bearer application-session' },
      { method: 'DELETE', url: '/v1/session', authorization: 'Bearer application-session' },
      { method: 'GET', url: '/v1/me', authorization: 'Bearer application-session' },
    ]);
    assert.deepEqual(JSON.parse(requests[0].body), { identityToken: 'provider-token' });
  } finally {
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()));
  }
});
