export function createRetryDeadline(now, totalTimeoutMilliseconds) {
  const deadline = now() + totalTimeoutMilliseconds
  const remaining = () => Math.max(0, deadline - now())

  return {
    canRetry: () => remaining() > 0,
    remaining,
    timeout: (maximum) => Math.max(1, Math.min(maximum, remaining())),
  }
}

export async function sendAndCaptureTryResponse(waitForResponse, send) {
  const responseOutcomePromise = waitForResponse().then(
    (response) => ({ response }),
    (error) => ({ error }),
  )
  const sent = await send()
  return { sent, ...await responseOutcomePromise }
}
