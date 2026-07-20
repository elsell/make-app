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

export function createSessionApiClient(baseUrl: string, tokenProvider: TokenProvider) {
  const client = createApiClient(baseUrl, tokenProvider);
  return {
    exchange: (identityToken: string) => client.POST('/v1/sessions', { body: { identityToken } }),
    refresh: () => client.POST('/v1/session/refresh'),
    profile: () => client.GET('/v1/me'),
    revoke: () => client.DELETE('/v1/session'),
  };
}
