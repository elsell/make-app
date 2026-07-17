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
  const authorizationRequest = await authorizationRequestPromise
  const authorizationURL = new URL(authorizationRequest.url())
  if (authorizationURL.searchParams.get('code_challenge_method') !== 'S256' || !authorizationURL.searchParams.get('code_challenge')) {
    throw new Error(`Scalar did not initiate S256 PKCE: ${authorizationURL}`)
  }
  const popup = await popupPromise
  await popup.locator('input[name=login]').fill(email)
  await popup.locator('input[name=password]').fill(password)

  const tokenRequestPromise = page.waitForRequest(
    (request) => request.url() === `${baseURL}/oidc/token`,
  )
  const tokenResponsePromise = page.waitForResponse(
    (response) => response.url() === `${baseURL}/oidc/token`,
  )
  await popup.getByRole('button', { name: 'Login' }).click()
  const tokenRequest = await tokenRequestPromise
  const tokenForm = new URLSearchParams(tokenRequest.postData() ?? '')
  if (tokenForm.get('grant_type') !== 'authorization_code' || !tokenForm.get('code_verifier')) {
    throw new Error(`Scalar omitted the PKCE verifier: ${tokenForm}`)
  }
  const tokenResponse = await tokenResponsePromise
  if (tokenResponse.status() !== 200) {
    throw new Error(`Scalar token exchange returned ${tokenResponse.status()}: ${await tokenResponse.text()}`)
  }
  for (let attempt = 0; attempt < 50 && !popup.isClosed(); attempt += 1) {
    await page.waitForTimeout(100)
  }
  if (!popup.isClosed()) {
    throw new Error('Scalar authorization popup did not close after token exchange')
  }

  async function tryRequest(buttonName, pathname) {
    await page.getByRole('button', { name: buttonName }).click()
    const responsePromise = page.waitForResponse(
      (response) => response.url().startsWith(`${baseURL}${pathname}`) && response.request().method() === 'GET',
    )
    await page.getByRole('button', { name: /Send Request/ }).click()
    const response = await responsePromise
    const authorization = await response.request().headerValue('authorization')
    if (!authorization?.startsWith('Bearer ')) {
      throw new Error(`Scalar omitted the OIDC bearer token for ${pathname}`)
    }
    if (response.status() !== 200) {
      throw new Error(`Scalar Try It ${pathname} returned ${response.status()}: ${await response.text()}`)
    }
    await page.getByRole('button', { name: 'Close Client' }).click()
    return response.json()
  }

  const me = await tryRequest(/Test Request.*\/v1\/me/, '/v1/me')
  if (me?.data?.email !== email) {
    throw new Error(`Scalar /v1/me returned the wrong principal: ${JSON.stringify(me)}`)
  }
  const resources = await tryRequest(/Test Request.*get \/v1\/examples\)/i, '/v1/examples')
  if (!Array.isArray(resources?.data)) {
    throw new Error(`Scalar resource list returned an invalid envelope: ${JSON.stringify(resources)}`)
  }
  console.log('Scalar browser OIDC and Try It acceptance passed')
} finally {
  await browser.close()
}
