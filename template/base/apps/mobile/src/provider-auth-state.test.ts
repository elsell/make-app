import assert from 'node:assert/strict';
import test from 'node:test';

import { classifyProviderResponse } from './provider-auth-state';

test('resolved provider errors surface a failed sign-in state', () => {
  assert.equal(classifyProviderResponse('error'), 'failed');
});

test('user cancellation and dismissal reset without reporting a provider failure', () => {
  assert.equal(classifyProviderResponse('cancel'), 'cancelled');
  assert.equal(classifyProviderResponse('dismiss'), 'cancelled');
});

test('success proceeds and incomplete browser states remain pending', () => {
  assert.equal(classifyProviderResponse('success'), 'success');
  assert.equal(classifyProviderResponse('opened'), 'pending');
  assert.equal(classifyProviderResponse('locked'), 'pending');
  assert.equal(classifyProviderResponse(undefined), 'pending');
});
