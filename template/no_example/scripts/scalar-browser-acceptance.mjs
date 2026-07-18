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
  await page.getByRole('button', { name: /Test Request.*get \/v1\/me\)/i }).click()
  const responsePromise = page.waitForResponse((response) => response.url() === `${baseURL}/v1/me`)
  await page.getByRole('button', { name: /Send Request/ }).click()
  const response = await responsePromise
  if (!(await response.request().headerValue('authorization'))?.startsWith('Bearer ') || response.status() !== 200) {
    throw new Error(`Scalar authenticated /v1/me request failed: ${response.status()}`)
  }
  console.log('Scalar browser OIDC and identity Try It acceptance passed')
} finally {
  await browser.close()
}
