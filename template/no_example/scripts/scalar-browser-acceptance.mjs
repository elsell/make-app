import { chromium } from 'playwright'

const baseURL = process.env.SCALAR_ACCEPTANCE_BASE_URL ?? 'http://localhost:8080'
const email = process.env.SCALAR_ACCEPTANCE_EMAIL ?? 'developer@example.com'
const password = process.env.SCALAR_ACCEPTANCE_PASSWORD ?? 'password'

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
    for (let attempt = 0; attempt < 20; attempt += 1) {
      await page.getByRole('button', { name: buttonName }).click()
      const responsePromise = page.waitForResponse(
        (response) => response.url().startsWith(`${baseURL}${pathname}`) && response.request().method() === 'GET',
      )
      await page.getByRole('button', { name: /Send Request/ }).click()
      const response = await responsePromise
      const authorization = await response.request().headerValue('authorization')
      if (authorization?.startsWith('Bearer ')) {
        if (response.status() !== 200) {
          throw new Error(`Scalar Try It ${pathname} returned ${response.status()}: ${await response.text()}`)
        }
        await page.getByRole('button', { name: 'Close Client' }).click()
        return response.json()
      }
      await page.getByRole('button', { name: 'Close Client' }).click()
      await page.waitForTimeout(250)
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
