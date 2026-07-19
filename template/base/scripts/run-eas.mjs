import { existsSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

export function runEAS(platform, options = {}) {
  if (!['android', 'ios'].includes(platform)) throw new Error('platform must be android or ios');
  const cwd = options.cwd ?? process.cwd();
  for (const name of ['app.json', 'eas.json']) {
    if (!existsSync(path.join(cwd, name))) throw new Error(`EAS must run from apps/mobile; missing ${name}`);
  }
  const executable = path.resolve(cwd, '../../tools/eas-cli/node_modules/.bin/eas');
  if (!existsSync(executable)) throw new Error('locked EAS CLI is not installed; run pnpm install --frozen-lockfile');
  const spawn = options.spawn ?? spawnSync;
  const result = spawn(executable, ['build', '--platform', platform, '--profile', 'production'], { cwd, stdio: 'inherit', env: process.env });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`EAS build failed with status ${result.status}`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) runEAS(process.argv[2]);
