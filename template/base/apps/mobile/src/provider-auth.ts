import * as AuthSession from 'expo-auth-session';
import * as WebBrowser from 'expo-web-browser';
import { useCallback, useEffect, useState } from 'react';

WebBrowser.maybeCompleteAuthSession();

export type ProviderSignIn = {
  ready: boolean;
  identityToken: string | null;
  failed: boolean;
  begin: () => Promise<void>;
  acknowledge: () => void;
};

export function useProviderSignIn(issuer: string, clientId: string, scheme: string): ProviderSignIn {
  const discovery = AuthSession.useAutoDiscovery(issuer);
  const redirectUri = AuthSession.makeRedirectUri({ scheme, path: 'callback' });
  const [request, response, prompt] = AuthSession.useAuthRequest({
    clientId,
    redirectUri,
    scopes: ['openid', 'profile', 'email'],
    usePKCE: true,
  }, discovery);
  const [identityToken, setIdentityToken] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (response?.type !== 'success' || !request?.codeVerifier || !discovery) return;
    void AuthSession.exchangeCodeAsync({
      clientId,
      code: response.params.code,
      redirectUri,
      extraParams: { code_verifier: request.codeVerifier },
    }, discovery).then((result) => {
      if (!result.idToken) throw new Error('identity_token_missing');
      setIdentityToken(result.idToken);
    }).catch(() => setFailed(true));
  }, [clientId, discovery, redirectUri, request?.codeVerifier, response]);

  const begin = useCallback(async () => {
    setFailed(false);
    setIdentityToken(null);
    try { await prompt(); }
    catch { setFailed(true); }
  }, [prompt]);
  const acknowledge = useCallback(() => {
    setFailed(false);
    setIdentityToken(null);
  }, []);
  return { ready: Boolean(request && discovery), identityToken, failed, begin, acknowledge };
}
