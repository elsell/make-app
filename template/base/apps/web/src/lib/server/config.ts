import { env } from '$env/dynamic/private';
import { parseWebConfig } from '$lib/config';

export function loadWebConfig() {
  return parseWebConfig({
    environment: env.__ENV_PREFIX___APP_ENV ?? 'development',
    apiURL: env.__ENV_PREFIX___API_URL ?? 'http://localhost:8080',
    oidcIssuer: env.__ENV_PREFIX___OIDC_ISSUER ?? 'http://localhost:5556/dex',
    oidcClientId: env.__ENV_PREFIX___WEB_OIDC_CLIENT_ID ?? '__APP_SLUG__-web',
  });
}

export const webConfig = loadWebConfig();
