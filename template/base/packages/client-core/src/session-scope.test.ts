import assert from 'node:assert/strict';
import test from 'node:test';
import { createSessionScope } from './session-scope.js';

test('account transitions invalidate old work even when the same account returns', () => {
  const scope = createSessionScope();
  scope.replace('alice');
  const alice = scope.capture();
  assert.equal(alice.current(), true);
  scope.replace('bob');
  assert.equal(alice.current(), false);
  scope.replace('alice');
  assert.equal(alice.current(), false);
  const renewed = scope.capture();
  scope.clear();
  assert.equal(renewed.current(), false);
  assert.equal(scope.capture().current(), false);
});
