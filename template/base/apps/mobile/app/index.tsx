import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as WebBrowser from 'expo-web-browser';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text } from 'react-native';
import { createApiClient } from '@__APP_SLUG__/api-client';

WebBrowser.maybeCompleteAuthSession();
const issuer = process.env.EXPO_PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
const clientId = process.env.EXPO_PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile';
const apiURL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';
const storageKey = 'oidc_tokens';

type Tokens = { accessToken: string; refreshToken?: string; issuedAt: number; expiresIn?: number };
type Profile = { id: string; email: string; displayName: string };

async function saveTokens(tokens: Tokens | null) {
  if (tokens) await SecureStore.setItemAsync(storageKey, JSON.stringify(tokens));
  else await SecureStore.deleteItemAsync(storageKey);
}

function expiring(tokens: Tokens) {
  return !tokens.expiresIn || tokens.issuedAt + tokens.expiresIn - 60 <= Date.now() / 1000;
}

export default function Home() {
  const [tokens, setTokens] = useState<Tokens | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const discovery = AuthSession.useAutoDiscovery(issuer);
  const redirectUri = AuthSession.makeRedirectUri({ scheme: '__APP_SLUG__', path: 'callback' });
  const [request, response, prompt] = AuthSession.useAuthRequest({ clientId, redirectUri, scopes: ['openid', 'profile', 'email', 'offline_access'], usePKCE: true }, discovery);

  async function activate(next: Tokens) {
    await saveTokens(next);
    setTokens(next);
    const result = await createApiClient(apiURL, () => next.accessToken).GET('/v1/me', { params: { header: { Authorization: `Bearer ${next.accessToken}` } } });
    if (result.error || !result.data?.data) throw new Error('The API rejected this session.');
    setProfile(result.data.data);
  }

  async function clearSession(message = '') {
    await saveTokens(null);
    setTokens(null);
    setProfile(null);
    setError(message);
  }

  useEffect(() => {
    if (!discovery) return;
    (async () => {
      try {
        const raw = await SecureStore.getItemAsync(storageKey);
        if (!raw) return;
        let stored = JSON.parse(raw) as Tokens;
        if (expiring(stored)) {
          if (!stored.refreshToken) throw new Error('Your session expired. Sign in again.');
          const refreshed = await AuthSession.refreshAsync({ clientId, refreshToken: stored.refreshToken }, discovery);
          stored = { accessToken: refreshed.accessToken, refreshToken: refreshed.refreshToken ?? stored.refreshToken, issuedAt: refreshed.issuedAt, expiresIn: refreshed.expiresIn };
        }
        await activate(stored);
      } catch (cause) {
        await clearSession(cause instanceof Error ? cause.message : 'Could not restore your session.');
      } finally { setReady(true); }
    })();
  }, [discovery]);

  useEffect(() => {
    if (response?.type !== 'success' || !request?.codeVerifier || !discovery) return;
    (async () => {
      try {
        const exchanged = await AuthSession.exchangeCodeAsync({ clientId, code: response.params.code, redirectUri, extraParams: { code_verifier: request.codeVerifier! } }, discovery);
        await activate({ accessToken: exchanged.accessToken, refreshToken: exchanged.refreshToken, issuedAt: exchanged.issuedAt, expiresIn: exchanged.expiresIn });
      } catch (cause) {
        await clearSession(cause instanceof Error ? cause.message : 'Sign-in failed.');
      } finally { setReady(true); }
    })();
  }, [response, request?.codeVerifier, discovery]);

  return <SafeAreaView style={{ padding: 32, gap: 16 }}>
    <Text style={{ fontSize: 32 }}>__APP_NAME__</Text>
    <Text>{!ready ? 'Loading…' : profile ? `Signed in as ${profile.displayName || profile.email}` : 'Ready to sign in'}</Text>
    {error ? <Text accessibilityRole="alert">{error}</Text> : null}
    <Button title={tokens ? 'Sign out' : 'Sign in'} disabled={!ready || (!tokens && !request)} onPress={() => tokens ? void clearSession() : void prompt()} />
  </SafeAreaView>;
}
