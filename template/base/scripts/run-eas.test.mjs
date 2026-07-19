import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { mkdtempSync } from 'node:fs';
import { runEAS } from './run-eas.mjs';

test('locked EAS runs from the Expo app and discovers both config files', () => {
  const root = mkdtempSync(path.join(tmpdir(), 'make-app-eas-'));
  const mobile = path.join(root, 'apps/mobile');
  const bin = path.join(root, 'tools/eas-cli/node_modules/.bin/eas');
  mkdirSync(path.dirname(bin), { recursive: true });
  mkdirSync(mobile, { recursive: true });
  writeFileSync(path.join(mobile, 'app.json'), '{}');
  writeFileSync(path.join(mobile, 'eas.json'), '{}');
  writeFileSync(bin, '');
  let invocation;
  runEAS('ios', { cwd: mobile, spawn: (command, args, options) => { invocation = { command, args, options }; return { status: 0 }; } });
  assert.equal(invocation.options.cwd, mobile);
  assert.equal(invocation.command, bin);
  assert.deepEqual(invocation.args, ['build', '--platform', 'ios', '--profile', 'production']);
});

test('locked EAS refuses a non-mobile working directory', () => {
  assert.throws(() => runEAS('android', { cwd: tmpdir(), spawn: () => ({ status: 0 }) }), /apps\/mobile/);
});
