import { UserManager, WebStorageStateStore } from 'oidc-client-ts';

function manager(issuer: string, clientId: string): UserManager {
  return new UserManager({
    authority: issuer,
    client_id: clientId,
    redirect_uri: `${window.location.origin}/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: 'code',
    scope: 'openid profile email',
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
    stateStore: new WebStorageStateStore({ store: window.sessionStorage }),
    automaticSilentRenew: false,
    monitorSession: false,
  });
}

export async function beginProviderSignIn(issuer: string, clientId: string): Promise<void> {
  await manager(issuer, clientId).signinRedirect();
}

export async function completeProviderSignIn(issuer: string, clientId: string): Promise<string> {
  const provider = manager(issuer, clientId);
  try {
    const user = await provider.signinRedirectCallback();
    if (!user.id_token) throw new Error('identity_token_missing');
    return user.id_token;
  } finally {
    await provider.removeUser().catch(() => undefined);
    await provider.clearStaleState().catch(() => undefined);
  }
}
