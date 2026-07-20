import assert from 'node:assert/strict';
import test from 'node:test';
import { activateExchangedSession } from './session-establishment';

const replacement = { token: 'replacement-session', expiresAt: '2026-07-21T12:00:00Z' };

for (const [status, discardCredential, state] of [
  [401, true, 'authentication_required'],
  [503, false, 'authenticated_offline'],
] as const) {
  test(`post-exchange profile ${status} ${discardCredential ? 'deletes' : 'retains'} the replacement credential`, async () => {
    const persisted: unknown[] = [];
    const outcome = await activateExchangedSession(
      replacement,
      async () => { throw { kind: 'http', status } as const; },
      async (credential) => { persisted.push(credential); },
    );
    assert.deepEqual(persisted, [discardCredential ? null : replacement]);
    assert.deepEqual(outcome?.decision, { state, discardCredential, retryable: status === 503 });
    assert.deepEqual(outcome?.failure, { kind: 'http', status });
  });
}
