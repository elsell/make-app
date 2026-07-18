# Configure a production OIDC provider

Create three public clients: `__APP_SLUG__-web`, `__APP_SLUG__-mobile`, and
`__APP_SLUG__-docs`. None receives a client secret.

Register these deployed redirect URIs:

- web: `https://APP_ORIGIN/callback`
- mobile: `__APP_NATIVE_ID__://callback`
- docs: `https://API_ORIGIN/docs`

Permit the web origin to perform the authorization-code token exchange. Enable
authorization code with S256 PKCE and the `openid`, `profile`, and `email`
scopes. Tokens must contain a stable subject, exact issuer, a configured client
audience, expiry, and a verified-email signal when invitation provisioning is
used. Multiple-audience tokens must include a valid authorized party.

Set the public issuer in every client and in `__ENV_PREFIX___OIDC_ISSUER`. Use
`__ENV_PREFIX___OIDC_BACKCHANNEL_URL` only when the API reaches the same issuer
through a private network address. Its path must match the public issuer; it
does not change token issuer validation.

Set `__ENV_PREFIX___OIDC_INSECURE=false` in production. Verify sign-in, session
exchange, refresh, revocation, account disablement, provider key rotation, and
provider unavailability before launch.
