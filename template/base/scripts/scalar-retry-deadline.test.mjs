import assert from 'node:assert/strict'
import test from 'node:test'

import { createRetryDeadline, sendAndCaptureTryResponse } from './scalar-retry-deadline.mjs'

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
