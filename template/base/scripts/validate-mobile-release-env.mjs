import assert from 'node:assert/strict';

assert.equal(process.env.__ENV_PREFIX___APP_ENV, 'production', 'signed mobile releases require __ENV_PREFIX___APP_ENV=production');
for (const name of ['__ENV_PREFIX___API_URL', '__ENV_PREFIX___OIDC_ISSUER', '__ENV_PREFIX___MOBILE_OIDC_CLIENT_ID']) {
  assert.ok(process.env[name], `${name} is required for a production mobile release`);
}
for (const [name, value] of [['__ENV_PREFIX___API_URL', process.env.__ENV_PREFIX___API_URL], ['__ENV_PREFIX___OIDC_ISSUER', process.env.__ENV_PREFIX___OIDC_ISSUER]]) {
  const parsed = new URL(value);
  const hostname = parsed.hostname.toLowerCase().replace(/^\[|\]$/g, '');
  const local = hostname === 'localhost' || hostname.endsWith('.localhost') || hostname === '::1' || hostname === '0.0.0.0' || hostname === '10.0.2.2' || hostname.startsWith('127.');
  assert.equal(parsed.protocol, 'https:', `${name} must use HTTPS`);
  assert.ok(!parsed.username && !parsed.password, `${name} must not contain URL credentials`);
  assert.ok(!parsed.search && !parsed.hash, `${name} must be a base URL without query or fragment`);
  assert.ok(!local, `${name} must not target a local or loopback host`);
}
console.log('production mobile release environment is explicit and safe');
