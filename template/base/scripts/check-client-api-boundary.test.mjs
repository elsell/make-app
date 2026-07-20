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
  'apps/web/src/lib/window-alias.ts': "const transport = window; transport.fetch('/v1/me');",
  'apps/web/src/lib/window-destructure.ts': "const { fetch: send } = window; send('/v1/me');",
  'apps/web/src/lib/document-default-view.ts': "const transport = document.defaultView; transport?.fetch('/v1/me');",
  'apps/web/src/lib/frames.ts': "const transport = frames[0]; transport.fetch('/v1/me');",
  'apps/web/src/lib/top.ts': "const transport = top; transport?.fetch('/v1/me');",
  'apps/web/src/lib/parent.ts': "const transport = parent; transport.fetch('/v1/me');",
  'apps/web/src/lib/opener-assignment.ts': "let transport; transport = opener; transport?.fetch('/v1/me');",
  'apps/web/src/lib/proxy-reassignment.ts': "let transport = window.location; transport = parent; transport.fetch('/v1/me');",
  'apps/web/src/lib/scoped-shadow.ts': "function shadow(document) { return document; } const transport = document.defaultView; transport?.fetch('/v1/me');",
  'apps/web/src/lib/before-shadow.ts': "const transport = window; function unrelated(window) { return window; } transport.fetch('/v1/me');",
  'apps/web/src/lib/sibling-shadow.ts': "{ const window = {}; void window; } const transport = window; transport.fetch('/v1/me');",
  'apps/web/src/lib/event-view.ts': "export function leak(event) { const transport = event.view; transport?.fetch('/v1/me'); }",
  'apps/web/src/lib/open-alias.ts': "const createWindow = open; const transport = createWindow('/child'); transport?.fetch('/v1/me');",
  'apps/web/src/lib/open-assignment.ts': "let createWindow; createWindow = open; const transport = createWindow('/child'); transport?.fetch('/v1/me');",
  'apps/web/src/lib/fetch-alias.ts': "const send = fetch; send('/v1/me');",
  'apps/web/src/lib/fetch-assignment.ts': "let send; send = fetch; send('/v1/me');",
  'apps/web/src/lib/xhr-alias.ts': "const Transport = XMLHttpRequest; new Transport();",
  'apps/web/src/lib/websocket-assignment.ts': "let Transport; Transport = WebSocket; new Transport('/v1/me');",
  'apps/web/src/lib/eventsource-alias.ts': "const Transport = EventSource; new Transport('/v1/me');",
  'apps/web/src/lib/request-assignment.ts': "let Transport; Transport = Request; new Transport('/v1/me');",
  'apps/web/src/lib/webtransport-alias.ts': "const Transport = WebTransport; new Transport('/v1/me');",
  'apps/web/src/lib/worker-assignment.ts': "let Transport; Transport = Worker; new Transport('/worker.js');",
  'apps/web/src/lib/shared-worker-alias.ts': "const Transport = SharedWorker; new Transport('/worker.js');",
  'apps/web/src/lib/rtc-assignment.ts': "let Transport; Transport = RTCPeerConnection; new Transport();",
  'apps/web/src/lib/beacon-alias.ts': "const send = sendBeacon; send('/v1/me');",
  'apps/web/src/lib/computed-primitive-key.ts': "const adapter = { [fetch]() { return 'network'; } }; adapter[fetch]();",
  'apps/web/src/lib/shorthand-primitive.ts': "const adapter = { fetch }; adapter.fetch('/v1/me');",
  'apps/mobile/src/beacon.ts': "navigator.sendBeacon(apiURL + '/v1/session');",
  'apps/mobile/src/dynamic.ts': "import('ax' + 'ios').then((transport) => transport.default(apiURL));",
  'apps/mobile/src/imported.ts': "import transport from 'openapi-fetch'; transport('/v1/me');",
  'apps/mobile/src/alias.ts': "import transport from '$shared/transport'; transport('/v1/me');",
  'apps/mobile/src/escaped.ts': "import { send } from '../../shared/transport'; send('/v1/me');",
  'apps/shared/transport.ts': "export const send = (url) => fetch(url);",
  'apps/mobile/src/same-dir.mjs': "export const send = (url) => fetch(url);",
  'apps/mobile/src/import-cjs.ts': "import { send } from './transport.cjs'; send('/v1/me');",
  'apps/mobile/src/transport.cjs': "exports.send = (url) => fetch(url);",
  'apps/web/src/lib/alias-cts.ts': "import { send } from '$lib/transport.cts'; send('/v1/me');",
  'apps/web/src/lib/transport.cts': "export const send = (url) => fetch(url);",
  'apps/web/src/lib/missing-alias.ts': "import { send } from '$lib/missing'; send('/v1/me');",
};
let result = check(bypasses);
assert.notEqual(result.status, 0, result.stderr);
for (const path of Object.keys(bypasses).filter((path) =>
  !path.startsWith('apps/shared/') &&
  path !== 'apps/mobile/src/import-cjs.ts' &&
  path !== 'apps/web/src/lib/transport.cts')) {
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

result = check({
  'apps/web/src/lib/local-primitives.ts': "function fetch(value) { return value; } class WebSocket {} const Request = (value) => value; fetch('local'); new WebSocket(); Request('local'); export function local(sendBeacon) { sendBeacon('local'); }",
  'apps/web/src/lib/block-local.ts': "const localHandler = (value) => value; { const fetch = localHandler; fetch('local'); }",
  'apps/web/src/lib/local-property-names.ts': "const adapter = { fetch() { return 'local'; }, get EventSource() { return 'local'; } }; class Local { Request() { return 'local'; } WebSocket = 'local'; } adapter.fetch(); new Local().Request();",
});
assert.equal(result.status, 0, result.stderr);

console.log('client API boundary rejects alternate transports and allows only exact provider adapters');
