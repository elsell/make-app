#!/usr/bin/env node
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';

const checker = resolve('scripts/check-client-api-boundary.mjs');
const protectedPaths = [
  'apps/mobile/src/provider-auth.ts',
  'apps/mobile/src/provider-auth-state.ts',
  'apps/web/src/lib/provider-auth.ts',
  'scripts/protected-provider-adapters.sha256',
];
const protectedBaseline = Object.fromEntries(protectedPaths.map((path) => [path, readFileSync(path, 'utf8')]));

function check(files) {
  const root = mkdtempSync(join(tmpdir(), 'client-api-boundary-'));
  try {
    for (const [relative, source] of Object.entries({ ...protectedBaseline, ...files })) {
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
  'apps/web/src/lib/ambient-const.ts': "declare const fetch: (url: string) => Promise<unknown>; const send = fetch; send('/v1/me');",
  'apps/web/src/lib/ambient-function.ts': "declare function fetch(url: string): Promise<unknown>; const send = fetch; send('/v1/me');",
  'apps/web/src/lib/ambient-class.ts': "declare class XMLHttpRequest { open(): void; } const Transport = XMLHttpRequest; new Transport();",
  'apps/web/src/lib/ambient-global.ts': "export {}; declare global { const fetch: (url: string) => Promise<unknown>; } const send = fetch; send('/v1/me');",
  'apps/web/src/lib/dotted-namespace.ts': "namespace Local.WebSocket { export const local = 'yes'; } new WebSocket('/v1/me');",
  'apps/web/src/lib/export-named.ts': "export { default as send } from 'openapi-fetch';",
  'apps/mobile/src/export-star.ts': "export * from 'openapi-fetch';",
  'apps/web/src/lib/import-equals.ts': "import transport = require('openapi-fetch'); transport('/v1/me');",
  'apps/mobile/src/require-alias.cjs': "const load = require; const transport = load('openapi-fetch'); transport('/v1/me');",
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
  'apps/mobile/app/index.tsx': "export default function Home() { return <></>; }",
  'apps/web/src/lib/generated.ts': "import { createSessionApiClient } from '@sample/api-client'; export const session = createSessionApiClient(base, token);",
});
assert.equal(result.status, 0, result.stderr);

result = check({
  'apps/mobile/src/provider-bypass.ts': "import * as AuthSession from 'expo-auth-session'; AuthSession.exchangeCodeAsync(options, discovery);",
  'apps/web/src/lib/provider-bypass.ts': "import { UserManager } from 'oidc-client-ts'; new UserManager(settings);",
});
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /provider-bypass\.ts/);

for (const [scenario, path, source] of [
  ['web-named', 'apps/web/src/lib/auth.ts', "export { UserManager as Provider } from 'oidc-client-ts';"],
  ['mobile-star', 'apps/mobile/app/index.tsx', "export * from 'expo-auth-session';"],
  ['web-namespace', 'apps/web/src/lib/auth.ts', "export * as Provider from 'oidc-client-ts';"],
  ['auth-import-export', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export { UserManager as Provider };"],
  ['provider-alias', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; const Provider = AuthSession; export { Provider };"],
  ['provider-variable', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; export const Provider = AuthSession;"],
  ['provider-factory', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export function createProvider() { return new UserManager(settings); }"],
  ['provider-factory-alias', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; function createProvider() { return new UserManager(settings); } export { createProvider };"],
  ['provider-object', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; export const Provider = { AuthSession };"],
  ['assignment-destructuring', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; let Provider; ({ AuthSession: Provider } = { AuthSession }); export { Provider };"],
  ['property-mutation', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; const exported = {}; exported.provider = AuthSession; export { exported };"],
  ['object-assign', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; const exported = {}; Object.assign(exported, { UserManager }); export { exported };"],
  ['returned-closure', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export const leak = () => () => UserManager;"],
  ['callback-argument', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export function leak(callback) { callback(UserManager); }"],
  ['exported-class-fields', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export class Leak { provider = UserManager; }"],
  ['default-class', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; export default class Home { provider = AuthSession; }"],
  ['identity-call', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; const identity = (value) => value; export const Leak = identity(UserManager);"],
  ['promise-call', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export const Leak = Promise.resolve(UserManager);"],
  ['logical-composite', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export const Leak = fallback || UserManager;"],
  ['comma-composite', 'apps/web/src/lib/auth.ts', "import { UserManager } from 'oidc-client-ts'; export const Leak = (fallback, UserManager);"],
  ['object-method', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; export const Leak = { provider() { return AuthSession; } };"],
  ['default-closure', 'apps/mobile/app/index.tsx', "import * as AuthSession from 'expo-auth-session'; export default () => AuthSession;"],
]) {
  const adapter = path.startsWith('apps/mobile/') ? 'apps/mobile/src/provider-auth.ts' : 'apps/web/src/lib/provider-auth.ts';
  result = check({ [adapter]: `${protectedBaseline[adapter]}\n// ${scenario}\n${source}` });
  assert.notEqual(result.status, 0, `${scenario}: ${result.stderr}`);
  assert.match(result.stderr, new RegExp(adapter.replaceAll('/', '\\/')));
}

for (const [scenario, source] of [
  ['async-home', 'export default async function Home() { return <></>; }'],
  ['generator-home', 'export default function* Home() { yield null; return <></>; }'],
]) {
  result = check({ 'apps/mobile/app/index.tsx': source });
  assert.notEqual(result.status, 0, `${scenario}: ${result.stderr}`);
  assert.match(result.stderr, /apps\/mobile\/app\/index\.tsx/);
}

result = check({ 'scripts/protected-provider-adapters.sha256': 'not a reviewed manifest\n' });
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /protected-provider-adapters\.sha256/);

result = check({
  'apps/mobile/src/provider-auth-state.ts': `${protectedBaseline['apps/mobile/src/provider-auth-state.ts']}\n// unreviewed classifier drift\n`,
});
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /apps\/mobile\/src\/provider-auth-state\.ts/);

result = check({ 'apps/web/src/lib/malformed.ts': "fetch('/v1/me'" });
assert.notEqual(result.status, 0, result.stderr);
assert.match(result.stderr, /malformed\.ts/);

result = check({
  'apps/web/src/lib/local-primitives.ts': "function fetch(value) { return value; } class WebSocket {} const Request = (value) => value; fetch('local'); new WebSocket(); Request('local'); export function local(sendBeacon) { sendBeacon('local'); }",
  'apps/web/src/lib/block-local.ts': "const localHandler = (value) => value; { const fetch = localHandler; fetch('local'); }",
  'apps/web/src/lib/local-property-names.ts': "const adapter = { fetch() { return 'local'; }, get EventSource() { return 'local'; } }; class Local { Request() { return 'local'; } WebSocket = 'local'; } adapter.fetch(); new Local().Request();",
  'apps/web/src/lib/runtime-type-bindings.ts': "enum Request { Local } namespace WebSocket { export const local = 'local'; } const request = Request.Local; const socket = WebSocket.local; export { request, socket };",
  'apps/web/src/lib/type-only-primitives.ts': "interface Request { local: string; } type EventSource = Request; const value: Request = { local: 'yes' }; export type { EventSource }; export { type Request, value };",
});
assert.equal(result.status, 0, result.stderr);

console.log('client API boundary rejects alternate transports and allows only exact provider adapters');
