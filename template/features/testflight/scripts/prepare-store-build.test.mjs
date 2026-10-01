import assert from 'node:assert/strict';
import test from 'node:test';
import { prepare } from './prepare-store-build.mjs';

test('store preparation sets versions and excludes development launcher', () => {
 const source={expo:{name:'Example',version:'1.0.0',ios:{bundleIdentifier:'com.example.app'},plugins:['expo-router','expo-dev-client',['expo-dev-client',{}]]}};
 const result=prepare(source,'v1.2.3','123');
 assert.equal(result.expo.version,'1.2.3');assert.equal(result.expo.ios.buildNumber,'123');
 assert.deepEqual(result.expo.plugins,['expo-router']);assert.equal(source.expo.version,'1.0.0');
 for(const tag of ['main','v01.2.3','v1.2.3-beta'])assert.throws(()=>prepare(source,tag,'123'));
 for(const build of ['0','1.100','1000000000'])assert.throws(()=>prepare(source,'v1.2.3',build));
});
