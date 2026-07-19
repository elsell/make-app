import { chromium, errors } from 'playwright'

const baseURL = process.env.SCALAR_ACCEPTANCE_BASE_URL ?? 'http://localhost:8080'
const email = process.env.SCALAR_ACCEPTANCE_EMAIL ?? 'developer@example.com'
const password = process.env.SCALAR_ACCEPTANCE_PASSWORD ?? 'password'
const webBaseURL = process.env.WEB_ACCEPTANCE_BASE_URL ?? 'http://localhost:5173'
const responseTimeoutMilliseconds = 5000

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

  async function waitForAuthorizedTryRequest(buttonName, pathname) {
    for (let attempt = 0; attempt < 20; attempt += 1) {
      await page.getByRole('button', { name: buttonName }).click()
      const responsePromise = page.waitForResponse(
        (response) => response.url().startsWith(`${baseURL}${pathname}`) && response.request().method() === 'GET',
        { timeout: responseTimeoutMilliseconds },
      )
      await page.getByRole('button', { name: /Send Request/ }).click()
      const response = await responsePromise.catch((error) => {
        if (error instanceof errors.TimeoutError) return null
        throw error
      })
      if (!response) {
        await page.getByRole('button', { name: 'Close Client' }).click()
        await page.waitForTimeout(250)
        continue
      }
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
  const resources = await waitForAuthorizedTryRequest(/Test Request.*get \/v1\/examples\)/i, '/v1/examples')
  if (!Array.isArray(resources?.data)) {
    throw new Error(`Scalar resource list returned an invalid envelope: ${JSON.stringify(resources)}`)
  }
  console.log('Scalar browser OIDC and Try It acceptance passed')

  const localizedContext = await browser.newContext({ locale: 'es-ES' })
  try {
    const localizedPage = await localizedContext.newPage()
    const localizedResponse = await localizedPage.goto(webBaseURL, { waitUntil: 'domcontentloaded' })
    const serverHTML = await localizedResponse?.text()
    if (!(localizedResponse?.headers()['vary'] ?? '').toLowerCase().split(',').map((value) => value.trim()).includes('accept-language')) {
      throw new Error('Locale-dependent web response omitted Vary: Accept-Language')
    }
    if (!serverHTML?.includes('<html lang="es">') || !serverHTML.includes('Cargando…')) {
      throw new Error('Web server did not render the negotiated Spanish locale before hydration')
    }
    await localizedPage.getByRole('button', { name: 'Iniciar sesión' }).waitFor()
    if (await localizedPage.locator('html').getAttribute('lang') !== 'es') {
      throw new Error('Web client did not select the supported Spanish base locale')
    }
    await localizedPage.getByText('Tu aplicación generada está lista.').waitFor()
    console.log('Web browser internationalization acceptance passed')

    await localizedPage.getByRole('button', { name: 'Iniciar sesión' }).click()
    await localizedPage.waitForURL((url) => url.origin === 'http://localhost:5556' && url.pathname.includes('/dex/auth/'))
    await localizedPage.locator('input[name=login]').fill(email)
    await localizedPage.locator('input[name=password]').fill(password)
    await localizedPage.getByRole('button', { name: 'Login' }).click()
    await localizedPage.waitForURL(`${webBaseURL}/`)
    await localizedPage.getByText(/^Sesión iniciada como /).waitFor()
    const browserExample = `Browser example ${Date.now()}`
    await localizedPage.getByLabel('Nombre del ejemplo').fill(browserExample)
    const createResponsePromise = localizedPage.waitForResponse(
      (response) => response.url() === `${baseURL}/v1/examples` && response.request().method() === 'POST',
    )
    await localizedPage.getByRole('button', { name: 'Crear ejemplo' }).click()
    const createResponse = await createResponsePromise
    if (createResponse.status() !== 201) {
      throw new Error(`Web example creation returned ${createResponse.status()}: ${await createResponse.text()}`)
    }
    await localizedPage.getByText(browserExample).waitFor()
    await localizedPage.getByText('Ejemplo creado.').waitFor()
    console.log('Web browser OIDC and application-session acceptance passed')
  } finally {
    await localizedContext.close()
  }

  for (const preference of [
    { header: 'es;q=0, en;q=1', lang: 'en', copy: 'Loading…' },
    { header: 'en;q=0.1, es;q=1', lang: 'es', copy: 'Cargando…' },
  ]) {
    const preferenceContext = await browser.newContext({ extraHTTPHeaders: { 'Accept-Language': preference.header } })
    try {
      const response = await preferenceContext.request.get(webBaseURL)
      const html = await response.text()
      if (!html.includes(`<html lang="${preference.lang}">`) || !html.includes(preference.copy)) {
        throw new Error(`Web server ignored Accept-Language quality values: ${preference.header}`)
      }
    } finally {
      await preferenceContext.close()
    }
  }
} finally {
  await browser.close()
}
