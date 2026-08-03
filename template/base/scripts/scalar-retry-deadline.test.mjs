import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createRetryDeadline,
  sendAndCaptureTryResponse,
  waitForStableCredential,
} from './scalar-retry-deadline.mjs'

test('rapid unauthenticated responses cannot exhaust the elapsed retry deadline early', () => {
  let now = 0
  const deadline = createRetryDeadline(() => now, 45_000)

  for (let attempt = 0; attempt < 20; attempt += 1) {
    assert.equal(deadline.canRetry(), true)
    now += 250
  }

  assert.equal(deadline.remaining(), 40_000)
  assert.equal(deadline.timeout(5_000), 5_000)
  now = 44_900
  assert.equal(deadline.timeout(5_000), 100)
  now = 45_000
  assert.equal(deadline.canRetry(), false)
  assert.equal(deadline.remaining(), 0)
})

test('credential readiness does not return on first equality', async () => {
  const pauses = []
  let now = 0
  await waitForStableCredential({
    now: () => now,
    matches: async () => true,
    pause: async (milliseconds) => { pauses.push(milliseconds); now += milliseconds },
  }, 'application-session', 1_000, 500, 100)

  assert.deepEqual(pauses, [100, 100, 100, 100, 100])
  assert.equal(now, 500)
})

test('a transient credential mismatch resets the stability interval', async () => {
  const matches = [true, false, true, true, true]
  const observations = []
  let now = 0
  await waitForStableCredential({
    now: () => now,
    matches: async () => {
      observations.push(now)
      return matches.shift() ?? false
    },
    pause: async (milliseconds) => { now += milliseconds },
  }, 'application-session', 125, 50, 25)

  assert.deepEqual(observations, [0, 25, 50, 75, 100])
  assert.equal(now, 100)
})

test('credential stability fails closed at its original monotonic deadline', async () => {
  let now = 0
  await assert.rejects(
    waitForStableCredential({
      now: () => now,
      matches: async () => true,
      pause: async (milliseconds) => { now += milliseconds },
    }, 'application-session', 60, 500, 25),
    /did not remain stable/,
  )
  assert.equal(now, 60)
})

test('a stalled credential observation receives and exhausts only the remaining deadline', async () => {
  const timeouts = []
  let now = 0
  await assert.rejects(
    waitForStableCredential({
      now: () => now,
      matches: async (_credential, timeout) => {
        timeouts.push(timeout)
        now += timeout
        return undefined
      },
      pause: async () => assert.fail('a timed-out observation must fail without another pause'),
    }, 'application-session', 60, 500, 25),
    /did not remain stable/,
  )
  assert.deepEqual(timeouts, [60])
  assert.equal(now, 60)
})

test('slow successful observations cannot replace the expected stability sequence', async () => {
  let now = 0
  let observations = 0
  await assert.rejects(
    waitForStableCredential({
      now: () => now,
      matches: async () => {
        observations += 1
        now += 200
        return true
      },
      pause: async (milliseconds) => { now += milliseconds },
    }, 'application-session', 1_000, 500, 50),
    /did not remain stable/,
  )
  assert.ok(observations < 11)
  assert.ok(now >= 1_000)
})

test('a timed-out send still consumes a captured response without rejecting the waiter', async () => {
  const events = []
  let resolveResponse
  const responsePromise = new Promise((resolve) => { resolveResponse = resolve })
  const resultPromise = sendAndCaptureTryResponse(
    () => {
      events.push('wait')
      return responsePromise
    },
    async () => {
      events.push('send')
      resolveResponse({ status: 401 })
      return false
    },
  )

  assert.deepEqual(await resultPromise, { sent: false, response: { status: 401 } })
  assert.deepEqual(events, ['wait', 'send'])
})

test('a response failure is captured while the send operation is pending', async () => {
  const responseFailure = new Error('page closed')
  let finishSend
  const resultPromise = sendAndCaptureTryResponse(
    () => Promise.reject(responseFailure),
    () => new Promise((resolve) => { finishSend = resolve }),
  )
  await Promise.resolve()
  finishSend(true)

  assert.deepEqual(await resultPromise, { sent: true, error: responseFailure })
})
