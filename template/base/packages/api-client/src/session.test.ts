import assert from 'node:assert/strict';
import test from 'node:test';
import { sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from './session.js';

const now = Date.parse('2026-07-17T00:00:00Z');

test('schedules ordinary refresh at the lead window', () => {
  assert.equal(sessionRefreshDelay('2026-07-17T01:00:00Z', now), 55 * 60 * 1000);
});

test('paces short sessions at their midpoint instead of every second', () => {
  assert.equal(sessionRefreshDelay('2026-07-17T00:04:00Z', now), 2 * 60 * 1000);
  assert.equal(sessionRefreshDelay(new Date(now + 500).toISOString(), now), 500);
  assert.equal(sessionRefreshDelay(new Date(now + sessionRefreshLeadMs).toISOString(), now), 150_000);
});

test('detects an absolute-deadline-capped replacement', () => {
  assert.equal(sessionExpiryAdvanced('2026-07-17T01:00:00Z', '2026-07-17T01:00:00Z'), false);
  assert.equal(sessionExpiryAdvanced('2026-07-17T01:00:00Z', '2026-07-17T02:00:00Z'), true);
});
