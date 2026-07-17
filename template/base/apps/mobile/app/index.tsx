import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as WebBrowser from 'expo-web-browser';
import { getLocales } from 'expo-localization';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text } from 'react-native';
import { createApiClient } from '@__APP_SLUG__/api-client';
import { type MessageKey } from '@__APP_SLUG__/i18n';
import { createDeviceTranslator } from '../src/i18n';

WebBrowser.maybeCompleteAuthSession();
const issuer = process.env.EXPO_PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
const clientId = process.env.EXPO_PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile';
const apiURL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';
const storageKey = 'oidc_tokens';
const i18n = createDeviceTranslator(getLocales);

type Tokens = { idToken: string; refreshToken?: string; issuedAt: number; expiresIn?: number };
type Profile = { id: string; email: string; displayName: string };

class LocalizedError extends Error {
  constructor(readonly key: MessageKey) { super(key); }
}

async function saveTokens(tokens: Tokens | null) {
  if (tokens) await SecureStore.setItemAsync(storageKey, JSON.stringify(tokens));
  else await SecureStore.deleteItemAsync(storageKey);
}

function expiring(tokens: Tokens) {
  return !tokens.expiresIn || tokens.issuedAt + tokens.expiresIn - 60 <= Date.now() / 1000;
}

function localizedFailure(cause: unknown, fallback: MessageKey): MessageKey {
  return cause instanceof LocalizedError ? cause.key : fallback;
}

export default function Home() {
  const [tokens, setTokens] = useState<Tokens | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [ready, setReady] = useState(false);
  const [errorKey, setErrorKey] = useState<MessageKey | null>(null);
  const discovery = AuthSession.useAutoDiscovery(issuer);
  const redirectUri = AuthSession.makeRedirectUri({ scheme: '__APP_SLUG__', path: 'callback' });
  const [request, response, prompt] = AuthSession.useAuthRequest({ clientId, redirectUri, scopes: ['openid', 'profile', 'email', 'offline_access'], usePKCE: true }, discovery);

  async function activate(next: Tokens) {
    await saveTokens(next);
    setTokens(next);
    const result = await createApiClient(apiURL, () => next.idToken).GET('/v1/me', { params: { header: { Authorization: `Bearer ${next.idToken}` } } });
    if (result.error || !result.data?.data) throw new LocalizedError('errors.apiRejected');
    setProfile(result.data.data);
  }

  async function clearSession(message: MessageKey | null = null) {
    await saveTokens(null);
    setTokens(null);
    setProfile(null);
    setErrorKey(message);
  }

  useEffect(() => {
    if (!discovery) return;
    (async () => {
      try {
        const raw = await SecureStore.getItemAsync(storageKey);
        if (!raw) return;
        let stored = JSON.parse(raw) as Tokens;
        if (expiring(stored)) {
          if (!stored.refreshToken) throw new LocalizedError('errors.sessionExpired');
          const refreshed = await AuthSession.refreshAsync({ clientId, refreshToken: stored.refreshToken }, discovery);
          if (!refreshed.idToken) throw new LocalizedError('errors.identityTokenRefreshMissing');
          stored = { idToken: refreshed.idToken, refreshToken: refreshed.refreshToken ?? stored.refreshToken, issuedAt: refreshed.issuedAt, expiresIn: refreshed.expiresIn };
        }
        await activate(stored);
      } catch (cause) {
        await clearSession(localizedFailure(cause, 'errors.restoreFailed'));
      } finally { setReady(true); }
    })();
  }, [discovery]);

  useEffect(() => {
    if (response?.type !== 'success' || !request?.codeVerifier || !discovery) return;
    (async () => {
      try {
        const exchanged = await AuthSession.exchangeCodeAsync({ clientId, code: response.params.code, redirectUri, extraParams: { code_verifier: request.codeVerifier! } }, discovery);
        if (!exchanged.idToken) throw new LocalizedError('errors.identityTokenMissing');
        await activate({ idToken: exchanged.idToken, refreshToken: exchanged.refreshToken, issuedAt: exchanged.issuedAt, expiresIn: exchanged.expiresIn });
      } catch (cause) {
        await clearSession(localizedFailure(cause, 'errors.signInFailed'));
      } finally { setReady(true); }
    })();
  }, [response, request?.codeVerifier, discovery]);

  return <SafeAreaView style={{ padding: 32, gap: 16 }}>
    <Text style={{ fontSize: 32 }}>{i18n.t('app.title')}</Text>
    <Text>{!ready ? i18n.t('common.loading') : profile ? i18n.t('auth.signedInAs', { name: profile.displayName || profile.email }) : i18n.t('auth.ready')}</Text>
    {errorKey ? <Text accessibilityRole="alert">{i18n.t(errorKey)}</Text> : null}
    <Button title={i18n.t(tokens ? 'auth.signOut' : 'auth.signIn')} disabled={!ready || (!tokens && !request)} onPress={() => tokens ? void clearSession() : void prompt()} />
  </SafeAreaView>;
}
