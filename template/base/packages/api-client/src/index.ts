import createClient from 'openapi-fetch';
import type { paths } from './schema';
export { sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from './session';

export type TokenProvider = () => Promise<string | null> | string | null;
export function createApiClient(baseUrl: string, tokenProvider: TokenProvider) {
  return createClient<paths>({
    baseUrl,
    fetch: async (input, init = {}) => {
      const token = await tokenProvider();
      const requestInit = init as RequestInit;
      const headers = new Headers(requestInit.headers);
      if (token) headers.set('Authorization', `Bearer ${token}`);
      return fetch(input, { ...requestInit, headers });
    }
  });
}
