export type TokenProvider = () => Promise<string | null> | string | null;

export function createAuthenticatedFetch(tokenProvider: TokenProvider): typeof fetch {
  return async (input, init = {}) => {
    const token = await tokenProvider();
    const headers = new Headers(input instanceof Request ? input.headers : undefined);
    new Headers(init.headers).forEach((value, key) => headers.set(key, value));
    headers.delete('Authorization');
    if (token) headers.set('Authorization', `Bearer ${token}`);
    return fetch(input, { ...init, headers });
  };
}
