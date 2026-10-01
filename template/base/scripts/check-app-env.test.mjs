import assert from 'node:assert/strict';
import test from 'node:test';
import { inspectApplicationEnvironmentSources, isApplicationEnvironmentSourcePath } from './check-app-env.mjs';

test('application configuration inventory requires the canonical prefix', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: 'process.env.__ENV_PREFIX___API_URL' },
  ]), []);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: 'process.env.PUBLIC_API_URL' },
  ]), ['apps/mobile/app.config.ts: PUBLIC_API_URL is not __ENV_PREFIX__-prefixed']);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: 'process.env.SURPRISE_ENDPOINT' },
  ]), ['apps/mobile/app.config.ts: SURPRISE_ENDPOINT is not __ENV_PREFIX__-prefixed']);
});

test('inventory permits only explicit tool-owned exceptions', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/web/Dockerfile', contents: 'ENV NODE_ENV=production HOST=0.0.0.0 PORT=3000' },
  ]), []);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/web/Dockerfile', contents: 'ENV NODE_ENV=production UNLISTED_TOOL_MODE=fast' },
  ]), ['apps/web/Dockerfile: UNLISTED_TOOL_MODE is not __ENV_PREFIX__-prefixed']);
});

test('inventory recognizes bracket, destructured, and Compose-list environment reads', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: "process.env['PUBLIC_API_URL']" },
    { path: 'apps/web/config.ts', contents: 'const { PUBLIC_OIDC_ISSUER } = process.env' },
    { path: 'compose.yaml', contents: 'environment:\n  - PUBLIC_OIDC_CLIENT_ID=__APP_SLUG__' },
  ]), [
    'apps/mobile/app.config.ts: PUBLIC_API_URL is not __ENV_PREFIX__-prefixed',
    'apps/web/config.ts: PUBLIC_OIDC_ISSUER is not __ENV_PREFIX__-prefixed',
    'compose.yaml: PUBLIC_OIDC_CLIENT_ID is not __ENV_PREFIX__-prefixed',
  ]);

  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: "const { __ENV_PREFIX___API_URL } = process.env; process.env['__ENV_PREFIX___OIDC_ISSUER']" },
    { path: 'compose.yaml', contents: 'environment:\n  - HOST=0.0.0.0' },
  ]), []);
});

test('inventory fails closed for computed process environment reads', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: 'process.env[name]' },
  ]), ['apps/mobile/app.config.ts: computed process.env access requires explicit review']);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: '// __APP_SLUG__-env-inventory: allow-computed reviewed fixed-key adapter\nprocess.env[name]' },
  ]), []);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/mobile/app.config.ts', contents: '// __APP_SLUG__-env-inventory: allow-computed reviewed fixed-key adapter\nprocess.env[first]\nprocess.env[second]' },
  ]), ['apps/mobile/app.config.ts: computed process.env access requires explicit review']);
});

test('inventory includes JavaScript-family and Go application sources', () => {
  for (const path of ['apps/web/config.js', 'apps/web/config.jsx', 'apps/web/config.mjs', 'apps/web/config.cjs', 'apps/api/internal/config/config.go']) {
    assert.equal(isApplicationEnvironmentSourcePath(path), true, path);
  }
  assert.equal(isApplicationEnvironmentSourcePath('apps/web/styles.css'), false);
});

test('inventory excludes only the exact reviewed vendored Scalar runtime', () => {
  assert.equal(isApplicationEnvironmentSourcePath('apps/api/internal/adapters/httpserver/assets/scalar-api-reference-1.44.20.js'), false);
  assert.equal(isApplicationEnvironmentSourcePath('/checkout/apps/api/internal/adapters/httpserver/assets/scalar-api-reference-1.44.20.js'), false);
  assert.equal(isApplicationEnvironmentSourcePath('apps/api/internal/adapters/httpserver/assets/scalar-api-reference-1.44.21.js'), true);
  assert.equal(isApplicationEnvironmentSourcePath('apps/api/internal/adapters/httpserver/assets/unreviewed.js'), true);
});

test('inventory recognizes literal Go environment reads and tool-owned exceptions', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: 'os.Getenv("PUBLIC_API_URL")' },
    { path: 'apps/api/cmd/tool/main.go', contents: 'os.LookupEnv("UNPREFIXED_MODE")' },
  ]), [
    'apps/api/cmd/tool/main.go: UNPREFIXED_MODE is not __ENV_PREFIX__-prefixed',
    'apps/api/internal/config/config.go: PUBLIC_API_URL is not __ENV_PREFIX__-prefixed',
  ]);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: 'os.Getenv("__ENV_PREFIX___API_URL"); os.LookupEnv("PATH")' },
  ]), []);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: 'database := env.DB; identity := env.ID' },
  ]), [], 'ordinary Go selectors are not environment reads');
});

test('inventory fails closed for computed Go environment reads unless reviewed', () => {
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: 'os.Getenv(name)' },
  ]), ['apps/api/internal/config/config.go: computed os environment access requires explicit review']);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: '// __APP_SLUG__-env-inventory: allow-computed reviewed typed configuration registry\nos.LookupEnv(name)' },
  ]), []);
  assert.deepEqual(inspectApplicationEnvironmentSources([
    { path: 'apps/api/internal/config/config.go', contents: '// __APP_SLUG__-env-inventory: allow-computed reviewed typed configuration registry\nos.LookupEnv(first)\nos.Getenv(second)' },
  ]), ['apps/api/internal/config/config.go: computed os environment access requires explicit review']);
});

test('inventory catches Compose interpolation and Svelte computed reads', () => {
  assert.ok(inspectApplicationEnvironmentSources([{path:'compose.yaml',contents:'command: ${UNREVIEWED_URL}'}]).length);
  assert.ok(inspectApplicationEnvironmentSources([{path:'apps/web/config.ts',contents:'env[name]'}]).length);
});
