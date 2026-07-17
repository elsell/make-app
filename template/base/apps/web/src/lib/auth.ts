import { UserManager, WebStorageStateStore } from 'oidc-client-ts';

export function createUserManager(): UserManager {
  const issuer = import.meta.env.PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
  const clientId = import.meta.env.PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-web';
  return new UserManager({
    authority: issuer,
    client_id: clientId,
    redirect_uri: `${window.location.origin}/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: 'code',
    scope: 'openid profile email offline_access',
    userStore: new WebStorageStateStore({ store: window.sessionStorage }),
    stateStore: new WebStorageStateStore({ store: window.sessionStorage }),
    automaticSilentRenew: true,
    monitorSession: false
  });
}
