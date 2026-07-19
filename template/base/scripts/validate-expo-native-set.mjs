import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const app = JSON.parse(await readFile(new URL('../apps/mobile/package.json', import.meta.url)));
const bundled = JSON.parse(await readFile(new URL('../apps/mobile/node_modules/expo/bundledNativeModules.json', import.meta.url)));
assert.deepEqual(app.expo?.install?.exclude, ['react-native'], 'only the documented React Native metadata exception is allowed');
assert.equal(app.dependencies['react-native'], bundled['react-native'], 'React Native must match Expo bundledNativeModules.json exactly');
console.log(`React Native ${app.dependencies['react-native']} matches the installed Expo native manifest`);
