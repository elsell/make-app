import createClient from 'openapi-fetch';
import type { paths } from './schema';
import { createAuthenticatedFetch } from './transport';
import type { TokenProvider } from './transport';
export type { TokenProvider } from './transport';
export { sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from './session';

export function createApiClient(baseUrl: string, tokenProvider: TokenProvider) {
  return createClient<paths>({
    baseUrl,
    fetch: createAuthenticatedFetch(tokenProvider),
  });
}
