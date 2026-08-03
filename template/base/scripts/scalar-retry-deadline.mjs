export function createRetryDeadline(now, totalTimeoutMilliseconds) {
  const deadline = now() + totalTimeoutMilliseconds
  const remaining = () => Math.max(0, deadline - now())

  return {
    canRetry: () => remaining() > 0,
    remaining,
    timeout: (maximum) => Math.max(1, Math.min(maximum, remaining())),
  }
}

export async function waitForStableCredential(
  port,
  expectedCredential,
  totalTimeoutMilliseconds,
  stabilityIntervalMilliseconds = 500,
  pollIntervalMilliseconds = 50,
) {
  const deadline = port.now() + totalTimeoutMilliseconds
  const requiredMatchingObservations = Math.ceil(stabilityIntervalMilliseconds / pollIntervalMilliseconds) + 1
  let stableSince = undefined
  let matchingObservations = 0

  while (port.now() < deadline) {
    const observationTimeout = deadline - port.now()
    const matches = await port.matches(expectedCredential, observationTimeout)
    const observedAt = port.now()
    if (matches === undefined) break
    if (matches) {
      stableSince ??= observedAt
      matchingObservations += 1
      if (
        observedAt <= deadline &&
        observedAt - stableSince >= stabilityIntervalMilliseconds &&
        matchingObservations >= requiredMatchingObservations
      ) return
    } else {
      stableSince = undefined
      matchingObservations = 0
    }

    const remaining = deadline - observedAt
    if (remaining > 0) await port.pause(Math.min(pollIntervalMilliseconds, remaining))
  }

  throw new Error('Scalar credential did not remain stable within the bounded readiness wait')
}

export async function sendAndCaptureTryResponse(waitForResponse, send) {
  const responseOutcomePromise = waitForResponse().then(
    (response) => ({ response }),
    (error) => ({ error }),
  )
  const sent = await send()
  return { sent, ...await responseOutcomePromise }
}
