import { chromium, errors } from 'playwright'
import { createRetryDeadline, sendAndCaptureTryResponse } from './scalar-retry-deadline.mjs'

const baseURL = process.env.SCALAR_ACCEPTANCE_BASE_URL ?? 'http://localhost:8080'
const email = process.env.SCALAR_ACCEPTANCE_EMAIL ?? 'developer@example.com'
const password = process.env.SCALAR_ACCEPTANCE_PASSWORD ?? 'password'
const responseTimeoutMilliseconds = 5000
const credentialApplicationTimeoutMilliseconds = 45_000

const browser = await chromium.launch({ headless: true })
try {
  const page = await browser.newPage()
  await page.goto(`${baseURL}/docs`, { waitUntil: 'domcontentloaded' })
  await page.getByRole('button', { name: /Authorize/ }).waitFor()
  const authorizationRequestPromise = page.context().waitForEvent('request', {
    predicate: (request) => new URL(request.url()).pathname.endsWith('/dex/auth'),
  })
  const popupPromise = page.waitForEvent('popup')
  await page.getByRole('button', { name: /Authorize/ }).click()
  const authorizationURL = new URL((await authorizationRequestPromise).url())
  if (authorizationURL.searchParams.get('code_challenge_method') !== 'S256' || !authorizationURL.searchParams.get('code_challenge')) {
    throw new Error(`Scalar did not initiate S256 PKCE: ${authorizationURL}`)
  }
  const popup = await popupPromise
  await popup.locator('input[name=login]').fill(email)
  await popup.locator('input[name=password]').fill(password)
  const tokenRequestPromise = page.waitForRequest((request) => request.url() === `${baseURL}/oidc/token`)
  const tokenResponsePromise = page.waitForResponse((response) => response.url() === `${baseURL}/oidc/token`)
  await popup.getByRole('button', { name: 'Login' }).click()
  const tokenForm = new URLSearchParams((await tokenRequestPromise).postData() ?? '')
  if (tokenForm.get('grant_type') !== 'authorization_code' || !tokenForm.get('code_verifier')) {
    throw new Error(`Scalar omitted the PKCE verifier: ${tokenForm}`)
  }
  const tokenResponse = await tokenResponsePromise
  if (tokenResponse.status() !== 200) {
    throw new Error(`Scalar token exchange returned ${tokenResponse.status()}: ${await tokenResponse.text()}`)
  }
  async function waitForAuthorizedTryRequest(buttonName, pathname) {
    const retryDeadline = createRetryDeadline(() => performance.now(), credentialApplicationTimeoutMilliseconds)
    const pauseBeforeRetry = async () => {
      const remaining = retryDeadline.remaining()
      if (remaining > 0) await page.waitForTimeout(Math.min(250, remaining))
    }
    const clickBeforeDeadline = async (locator) => {
      if (!retryDeadline.canRetry()) return false
      return locator.click({ timeout: retryDeadline.timeout(responseTimeoutMilliseconds) }).then(
        () => true,
        (error) => {
          if (error instanceof errors.TimeoutError) return false
          throw error
        },
      )
    }
    while (retryDeadline.canRetry()) {
      if (!await clickBeforeDeadline(page.getByRole('button', { name: buttonName }))) {
        await pauseBeforeRetry()
        continue
      }
      const responseOutcome = await sendAndCaptureTryResponse(
        () => page.waitForResponse(
          (response) => response.url().startsWith(`${baseURL}${pathname}`) && response.request().method() === 'GET',
          { timeout: retryDeadline.timeout(responseTimeoutMilliseconds) },
        ),
        () => clickBeforeDeadline(page.getByRole('button', { name: /Send Request/ })),
      )
      if (responseOutcome.error && !(responseOutcome.error instanceof errors.TimeoutError)) throw responseOutcome.error
      const response = responseOutcome.response ?? null
      if (!response) {
        await clickBeforeDeadline(page.getByRole('button', { name: 'Close Client' }))
        await pauseBeforeRetry()
        continue
      }
      const authorization = await response.request().headerValue('authorization')
      if (authorization?.startsWith('Bearer ')) {
        if (response.status() !== 200) {
          throw new Error(`Scalar Try It ${pathname} returned ${response.status()}: ${await response.text()}`)
        }
        await clickBeforeDeadline(page.getByRole('button', { name: 'Close Client' }))
        return response.json()
      }
      await clickBeforeDeadline(page.getByRole('button', { name: 'Close Client' }))
      await pauseBeforeRetry()
    }
    throw new Error(`Scalar omitted the OIDC bearer token for ${pathname} after the bounded credential-application wait`)
  }

  const me = await waitForAuthorizedTryRequest(/Test Request.*get \/v1\/me\)/i, '/v1/me')
  if (me?.data?.email !== email) {
    throw new Error(`Scalar /v1/me returned the wrong principal: ${JSON.stringify(me)}`)
  }
  console.log('Scalar browser OIDC and identity Try It acceptance passed')
} finally {
  await browser.close()
}
