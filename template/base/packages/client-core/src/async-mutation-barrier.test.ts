import assert from 'node:assert/strict';
import test from 'node:test';

import { createAsyncMutationBarrier } from './async-mutation-barrier.js';

test('blocking waits for every admitted mutation and rejects newer work', async () => {
  const barrier = createAsyncMutationBarrier();
  const first = barrier.enter();
  const second = barrier.enter();
  assert.ok(first);
  assert.ok(second);

  let drained = false;
  const waiting = barrier.blockAndDrain().then(() => { drained = true; });
  await Promise.resolve();
  assert.equal(drained, false);
  assert.equal(barrier.enter(), null);

  first.release();
  await Promise.resolve();
  assert.equal(drained, false);
  second.release();
  await waiting;
  assert.equal(drained, true);
});

test('leases release once and an explicit unblock admits the next mutation', async () => {
  const barrier = createAsyncMutationBarrier();
  const lease = barrier.enter();
  assert.ok(lease);
  lease.release();
  lease.release();
  await barrier.blockAndDrain();
  assert.equal(barrier.enter(), null);
  barrier.unblock();
  assert.ok(barrier.enter());
});

test('a sign-out snapshot taken after draining includes a start acknowledged in flight', async () => {
  const barrier = createAsyncMutationBarrier();
  const runningPathIDs = new Set<string>();
  const lease = barrier.enter();
  assert.ok(lease);
  let acknowledgeStart!: () => void;
  const startAcknowledgement = new Promise<void>((resolve) => { acknowledgeStart = resolve; });
  const start = startAcknowledgement.then(() => {
    runningPathIDs.add('path-starting');
    lease.release();
  });

  const drain = barrier.blockAndDrain();
  assert.equal(barrier.enter(), null);
  acknowledgeStart();
  await Promise.all([start, drain]);

  assert.deepEqual([...runningPathIDs], ['path-starting']);
});
