import { createSessionApiClient } from '@__APP_SLUG__/api-client';
import { refreshSessionCredential, sessionCredentialFromResponse, type ClientRuntimeConfig } from '@__APP_SLUG__/client-core';
import { beginProviderSignIn, completeProviderSignIn } from './provider-auth';

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
export async function exchangeApplicationSession(identityToken: string, config: ClientRuntimeConfig): Promise<void> {
  const result = await createSessionApiClient(config.apiURL, () => null).exchange(identityToken);
  saveApplicationSession(sessionCredentialFromResponse(result));
}
export async function refreshApplicationSession(config: ClientRuntimeConfig): Promise<ApplicationSession> {
  const current = applicationSession();
  if (!current) throw { kind: 'local_storage', reason: 'missing_fields' } as const;
  return refreshSessionCredential(
    current,
    async (credential) => createSessionApiClient(config.apiURL, () => credential.token).refresh(),
    async (replacement) => saveApplicationSession(replacement),
  );
}
export async function revokeApplicationSession(config: ClientRuntimeConfig): Promise<void> {
	const session = applicationSession();
	clearApplicationSession();
	if (session) {
		try { await createSessionApiClient(config.apiURL, () => session.token).revoke(); }
    catch { /* Local credential disposal must not depend on network availability. */ }
  }
}

export async function beginApplicationSignIn(config: ClientRuntimeConfig): Promise<void> {
  await beginProviderSignIn(config.oidcIssuer, config.oidcClientId);
}

export async function completeApplicationSignIn(config: ClientRuntimeConfig): Promise<void> {
  await exchangeApplicationSession(await completeProviderSignIn(config.oidcIssuer, config.oidcClientId), config);
}
