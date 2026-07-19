import assert from 'node:assert/strict';
import test from 'node:test';
import { classifySessionFailure, isSessionFailure, type SessionAccessState } from '@__APP_SLUG__/client-core';
import { restoreStoredSession } from './session-restoration';

test('cold launch restores a valid session offline with OIDC discovery unavailable and API unavailable', async () => {
  let state: SessionAccessState = 'authentication_required';
  let retainedToken = '';
  let discarded = false;
  let refreshCalled = false;
  await restoreStoredSession({
    read: async () => JSON.stringify({ token: 'stored-token', expiresAt: '2026-07-20T12:00:00Z' }),
    now: () => Date.parse('2026-07-19T12:00:00Z'),
    refreshLeadMs: 60_000,
    refresh: async (current) => { refreshCalled = true; return current; },
    expiryAdvanced: () => true,
    activate: async () => { throw { kind: 'network' } as const; },
    handleFailure: async (cause, current) => {
      const decision = classifySessionFailure(isSessionFailure(cause) ? cause : { kind: 'network' });
      state = decision.state;
      retainedToken = current.token;
      discarded = decision.discardCredential;
    },
    handleUnreadable: async () => { state = 'local_session_unreadable'; discarded = true; },
  });
  assert.equal(state, 'authenticated_offline');
  assert.equal(retainedToken, 'stored-token');
  assert.equal(discarded, false);
  assert.equal(refreshCalled, false);
});
