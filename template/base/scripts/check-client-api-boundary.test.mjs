#!/usr/bin/env node
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const checker = resolve('scripts/check-client-api-boundary.mjs');

function check(files) {
  const root = mkdtempSync(join(tmpdir(), 'client-api-boundary-'));
  try {
    for (const [relative, source] of Object.entries(files)) {
      const path = join(root, relative);
      mkdirSync(dirname(path), { recursive: true });
      writeFileSync(path, source);
    }
    return spawnSync(process.execPath, [checker, root], { encoding: 'utf8' });
  } finally { rmSync(root, { recursive: true, force: true }); }
}

const bypasses = {
  'apps/web/src/lib/direct.ts': "fetch(apiURL + '/v1/me');",
  'apps/web/src/lib/computed.ts': "(globalThis as any)['fe' + 'tch'](apiURL + '/v1/me');",
  'apps/web/src/lib/window.ts': "window['fetch'](apiURL + '/v1/me');",
  'apps/mobile/src/beacon.ts': "navigator.sendBeacon(apiURL + '/v1/session');",
  'apps/mobile/src/dynamic.ts': "import('ax' + 'ios').then((transport) => transport.default(apiURL));",
  'apps/mobile/src/imported.ts': "import transport from 'openapi-fetch'; transport('/v1/me');",
  'apps/mobile/src/alias.ts': "import transport from '$shared/transport'; transport('/v1/me');",
  'apps/mobile/src/escaped.ts': "import { send } from '../../shared/transport'; send('/v1/me');",
  'apps/shared/transport.ts': "export const send = (url) => fetch(url);",
  'apps/mobile/src/same-dir.mjs': "export const send = (url) => fetch(url);",
};
let result = check(bypasses);
assert.notEqual(result.status, 0, result.stderr);
for (const path of Object.keys(bypasses).filter((path) => !path.startsWith('apps/shared/'))) {
  assert.match(result.stderr, new RegExp(path.replaceAll('/', '\\/')));
}

result = check({
  'packages/api-client/package.json': '{"name":"@sample/api-client"}',
  'apps/mobile/app/index.tsx': "import * as AuthSession from 'expo-auth-session'; import * as WebBrowser from 'expo-web-browser'; AuthSession.exchangeCodeAsync(options, discovery); WebBrowser.maybeCompleteAuthSession();",
  'apps/web/src/lib/auth.ts': "import { UserManager } from 'oidc-client-ts'; export const manager = new UserManager(settings);",
  'apps/web/src/lib/generated.ts': "import { createSessionApiClient } from '@sample/api-client'; export const session = createSessionApiClient(base, token);",
});
assert.equal(result.status, 0, result.stderr);

result = check({
  'apps/mobile/src/provider-bypass.ts': "import * as AuthSession from 'expo-auth-session'; AuthSession.exchangeCodeAsync(options, discovery);",
  'apps/web/src/lib/provider-bypass.ts': "import { UserManager } from 'oidc-client-ts'; new UserManager(settings);",
});
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /provider-bypass\.ts/);

result = check({ 'apps/web/src/lib/malformed.ts': "fetch('/v1/me'" });
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /malformed\.ts/);

console.log('client API boundary rejects alternate transports and allows only exact provider adapters');
