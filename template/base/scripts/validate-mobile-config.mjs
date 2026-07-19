import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const config = JSON.parse(await readFile(new URL('../apps/mobile/app.json', import.meta.url)));
const eas = JSON.parse(await readFile(new URL('../apps/mobile/eas.json', import.meta.url)));
const app = config.expo;
assert.match(app.scheme, /^[a-z][a-z0-9]*$/, 'mobile callback scheme must be a safe native identifier');
assert.equal(app.ios.bundleIdentifier, `${app.ios.bundleIdentifier.slice(0, -app.scheme.length)}${app.scheme}`);
assert.equal(app.android.package, app.ios.bundleIdentifier, 'iOS and Android identifiers must share the rendered application identity');
assert.match(app.version, /^\d+\.\d+\.\d+$/, 'application version must be semantic');
assert.match(app.ios.buildNumber, /^\d+$/, 'iOS build number must be numeric');
assert.ok(Number.isSafeInteger(app.android.versionCode) && app.android.versionCode > 0, 'Android version code must be positive');
assert.deepEqual(app.runtimeVersion, { policy: 'appVersion' });
assert.equal(eas.build.development.developmentClient, true);
assert.equal(eas.build.development.env.APP_ENV, 'development');
assert.equal(eas.build.production.env.APP_ENV, 'production');
assert.equal(eas.build.production.environment, 'production');
assert.equal(eas.build.production.env.EXPO_PUBLIC_APP_ENV, 'production');
for (const path of [app.icon, app.splash.image, app.android.adaptiveIcon.foregroundImage, app.web.favicon]) {
  assert.match(path, /^\.\/assets\//, `mobile asset must be repository-owned: ${path}`);
}
console.log('mobile identifiers and release configuration are consistent');
