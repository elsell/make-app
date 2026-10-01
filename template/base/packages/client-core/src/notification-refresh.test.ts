import assert from 'node:assert/strict';
import test from 'node:test';

import { createNotificationRefreshLatch } from './notification-refresh.js';

test('notification refreshes serialize, coalesce, and stop after disposal', async () => {
  const releases: Array<() => void> = [];
  const owners: string[] = [];
  let active = 0;
  let maximumActive = 0;
  const latch = createNotificationRefreshLatch(async (ownerId) => {
    owners.push(ownerId);
    active += 1;
    maximumActive = Math.max(maximumActive, active);
    await new Promise<void>((resolve) => releases.push(resolve));
    active -= 1;
  });

  const first = latch.request('user-a');
  const coalesced = latch.request('user-a');
  assert.equal(first, coalesced);
  assert.deepEqual(owners, ['user-a']);
  releases.shift()?.();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(owners, ['user-a', 'user-a']);
  releases.shift()?.();
  await first;
  assert.equal(maximumActive, 1);

  latch.dispose();
  await latch.request('user-a');
  assert.deepEqual(owners, ['user-a', 'user-a']);
});

test('a mutation queued behind a stale refresh receives a trailing authoritative refresh', async () => {
  let serverRead = false;
  let visibleRead = false;
  let releaseFirst: (() => void) | undefined;
  let refreshCount = 0;
  let active = 0;
  let maximumActive = 0;
  const latch = createNotificationRefreshLatch(async () => {
    active += 1;
    maximumActive = Math.max(maximumActive, active);
    refreshCount += 1;
    const admittedRead = serverRead;
    if (refreshCount === 1) {
      await new Promise<void>((resolve) => { releaseFirst = resolve; });
    }
    visibleRead = admittedRead;
    active -= 1;
  });

  const initialRefresh = latch.request('user-a');
  await new Promise((resolve) => setTimeout(resolve, 0));
  serverRead = true;
  visibleRead = true;
  const mutationRefresh = latch.request('user-a');
  releaseFirst?.();
  await mutationRefresh;

  assert.equal(initialRefresh, mutationRefresh);
  assert.equal(refreshCount, 2);
  assert.equal(maximumActive, 1);
  assert.equal(visibleRead, true);
});

test('failed refresh does not lose a queued account refresh', async () => {
  let reject!: (error: Error) => void;
  const owners: string[] = [];
  const latch = createNotificationRefreshLatch(async owner => {
    owners.push(owner);
    if (owner === 'old') await new Promise<void>((_, fail) => { reject = fail; });
  });
  const first = latch.request('old');
  const trailing = latch.request('new');
  reject(new Error('offline'));
  await trailing;
  await first;
  assert.deepEqual(owners, ['old', 'new']);
});

test('disposal invalidates the callback lease before a late result commits', async () => {
  let finish!: () => void;
  let writes = 0;
  const latch = createNotificationRefreshLatch(async (_owner, current) => {
    await new Promise<void>(resolve => { finish = resolve; });
    if (current()) writes++;
  });
  const work = latch.request('owner');
  latch.dispose(); finish(); await work;
  assert.equal(writes, 0);
});

 test('synchronously reentrant requests share the active drain', async () => {
  const owners: string[] = [];
  let nested: Promise<void> | undefined;
  const latch = createNotificationRefreshLatch(async owner => {
    owners.push(owner);
    if (owner === 'first') nested = latch.request('second');
  });
  const first = latch.request('first');
  assert.equal(nested, first);
  await first;
  assert.deepEqual(owners, ['first', 'second']);
});

test('a request at drain completion starts a fresh authoritative read', async () => {
 const owners: string[] = [];
 const latch = createNotificationRefreshLatch(async owner => { owners.push(owner); });
 const first = latch.request('first');
 await Promise.resolve();
 const second = latch.request('second');
 await Promise.all([first, second]);
 assert.deepEqual(owners, ['first', 'second']);
});
