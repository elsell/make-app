import { UserManager, WebStorageStateStore } from 'oidc-client-ts';
import { isValidSessionCredential, refreshSessionCredential } from '@__APP_SLUG__/client-core';
import { apiURL, oidcClientId, oidcIssuer } from '$lib/config';

const sessionKey = '__APP_SLUG___application_session';
export type ApplicationSession = { token: string; expiresAt: string };

export function applicationSession(): ApplicationSession | null {
  const raw = window.sessionStorage.getItem(sessionKey);
  if (!raw) return null;
  try {
    const value = JSON.parse(raw) as ApplicationSession;
    return value.token && Number.isFinite(Date.parse(value.expiresAt)) ? value : null;
  } catch { return null; }
}
export function clearApplicationSession(): void { window.sessionStorage.removeItem(sessionKey); }
function saveApplicationSession(session: ApplicationSession): void { window.sessionStorage.setItem(sessionKey, JSON.stringify(session)); }
export async function exchangeApplicationSession(identityToken: string): Promise<void> {
  const headers = new Headers(); headers.set('Content-Type', 'application/json');
  const response = await fetch(`${apiURL}/v1/sessions`, { method: 'POST', headers, body: JSON.stringify({ identityToken }) });
  if (!response.ok) throw new Error('session_exchange_failed');
  const body = await response.json() as { data?: ApplicationSession };
  if (!isValidSessionCredential(body.data)) throw new Error('session_exchange_failed');
  saveApplicationSession(body.data);
}
export async function refreshApplicationSession(): Promise<ApplicationSession> {
  const current = applicationSession();
  if (!current) throw { kind: 'local_storage', reason: 'missing_fields' } as const;
  return refreshSessionCredential(
    current,
    async (credential) => fetch(`${apiURL}/v1/session/refresh`, { method: 'POST', headers: { Authorization: `Bearer ${credential.token}` } }),
    async (replacement) => saveApplicationSession(replacement),
  );
}
export async function revokeApplicationSession(): Promise<void> {
	const session = applicationSession();
	clearApplicationSession();
	if (session) {
		try { await fetch(`${apiURL}/v1/session`, { method: 'DELETE', headers: { Authorization: `Bearer ${session.token}` } }); }
    catch { /* Local credential disposal must not depend on network availability. */ }
  }
}

export function createUserManager(): UserManager {
  return new UserManager({
    authority: oidcIssuer,
    client_id: oidcClientId,
    redirect_uri: `${window.location.origin}/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: 'code',
    scope: 'openid profile email',
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
    stateStore: new WebStorageStateStore({ store: window.sessionStorage }),
    automaticSilentRenew: false,
    monitorSession: false
  });
}
