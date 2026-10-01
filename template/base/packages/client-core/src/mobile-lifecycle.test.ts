import assert from 'node:assert/strict';
import test from 'node:test';
import { createPullLifecycle, createActionRegistry } from './mobile-lifecycle.js';

test('pull lifecycle ends on navigation and ignores old completion', async () => {
 const states: boolean[]=[];const pull=createPullLifecycle(value=>states.push(value));
 let finish!:()=>void;let next!:()=>void;
 pull.focus();const old=pull.run(()=>new Promise(resolve=>{finish=resolve}));
 await pull.run(async()=>{throw new Error('duplicate should not execute')});
 pull.blur();pull.focus();const current=pull.run(()=>new Promise(resolve=>{next=resolve}));
 finish();await old;assert.equal(states.at(-1),true);
 next();await current;assert.deepEqual(states,[true,false,true,false]);
 await assert.rejects(pull.run(async()=>{throw new Error('failure')}));assert.equal(states.at(-1),false);
});
test('native commands invoke current committed action and ignore disabled or removed actions', () => {
 const registry=createActionRegistry();const called:string[]=[];const press=registry.handler('save');
 registry.commit([{id:'save',onPress:()=>called.push('old')}]);
 registry.commit([{id:'save',onPress:()=>called.push('current')}]);press();
 registry.commit([{id:'save',disabled:true,onPress:()=>called.push('disabled')}]);press();
 registry.commit([]);press();assert.deepEqual(called,['current']);
 assert.throws(()=>registry.commit([{id:'save',onPress:()=>{}},{id:'save',onPress:()=>{}}]));
});
