import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as WebBrowser from 'expo-web-browser';
import Constants from 'expo-constants';
import { getLocales } from 'expo-localization';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text } from 'react-native';
import { sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
import { classifySessionFailure, isSessionFailure, isValidSessionCredential, refreshSessionCredential, sessionFailureFromResponse, sessionRetryDelay, validateSessionCredential, type SessionAccessState, type SessionFailure } from '@__APP_SLUG__/client-core';
import { type MessageKey } from '@__APP_SLUG__/i18n';
import { createDeviceTranslator } from '../src/i18n';
import { loadMobileConfig } from '../src/config';

WebBrowser.maybeCompleteAuthSession();
const { apiURL, oidcClientId: clientId, oidcIssuer: issuer } = loadMobileConfig(Constants.expoConfig?.extra);
const storageKey = 'application_session';
const i18n = createDeviceTranslator(getLocales);

type Session = { token: string; expiresAt: string };
type Profile = { id: string; email: string; displayName: string };
class LocalizedError extends Error { constructor(readonly key: MessageKey) { super(key); } }

async function saveSession(session: Session | null) {
  if (session) await SecureStore.setItemAsync(storageKey, JSON.stringify(session));
  else await SecureStore.deleteItemAsync(storageKey);
}
async function refreshSession(session: Session): Promise<Session> {
  return refreshSessionCredential(
    session,
    async (current) => fetch(`${apiURL}/v1/session/refresh`, { method: 'POST', headers: { Authorization: `Bearer ${current.token}` } }),
    async (replacement) => saveSession(replacement),
  );
}
function localizedFailure(cause: unknown, fallback: MessageKey): MessageKey { return cause instanceof LocalizedError ? cause.key : fallback; }

export default function Home() {
  const [session, setSession] = useState<Session | null>(null);
  const [accessState, setAccessState] = useState<SessionAccessState>('authentication_required');
  const [sessionRenewable, setSessionRenewable] = useState(true);
  const [retryAttempt, setRetryAttempt] = useState(0);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [ready, setReady] = useState(false);
  const [errorKey, setErrorKey] = useState<MessageKey | null>(null);
  const discovery = AuthSession.useAutoDiscovery(issuer);
  const redirectUri = AuthSession.makeRedirectUri({ scheme: '__APP_NATIVE_ID__', path: 'callback' });
  const [request, response, prompt] = AuthSession.useAuthRequest({ clientId, redirectUri, scopes: ['openid', 'profile', 'email'], usePKCE: true }, discovery);

  async function activate(next: Session, renewable = true) {
    await saveSession(next); setSessionRenewable(renewable); setSession(next);
    const nextProfile = await validateSessionCredential<Profile>(next, async (current) =>
      fetch(`${apiURL}/v1/me`, { headers: { Authorization: `Bearer ${current.token}` } }),
    );
    setProfile(nextProfile);
    setAccessState('authenticated_online');
    setRetryAttempt(0);
  }
  async function clearSession(message: MessageKey | null = null, state: SessionAccessState = 'authentication_required') {
    const token = session?.token;
    await saveSession(null); setSession(null); setRetryAttempt(0); setProfile(null); setAccessState(state); setErrorKey(message);
    if (token) try { await fetch(`${apiURL}/v1/session`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } }); } catch { /* local disposal is authoritative */ }
  }
  async function handleSessionFailure(cause: unknown, current: Session) {
    const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
    const decision = classifySessionFailure(failure);
    if (decision.discardCredential) {
      await clearSession(decision.state === 'local_session_unreadable' ? 'errors.localSessionUnreadable' : 'errors.sessionExpired', decision.state);
      return;
    }
    await saveSession(current); setSession(current); setAccessState('authenticated_offline');
    if (!decision.retryable) { setSessionRenewable(false); setRetryAttempt(0); }
    if (decision.retryable) setRetryAttempt((attempt) => attempt + 1);
    setErrorKey(decision.retryable ? 'errors.temporarilyUnavailable' : 'errors.apiRejected');
  }

  useEffect(() => {
    if (!discovery) return;
    (async () => {
      try {
        const raw = await SecureStore.getItemAsync(storageKey); if (!raw) return;
        let stored: Session;
        try { stored = JSON.parse(raw) as Session; } catch { throw { kind: 'local_storage', reason: 'malformed' } satisfies SessionFailure; }
        if (!stored.token || !stored.expiresAt || !Number.isFinite(Date.parse(stored.expiresAt))) throw { kind: 'local_storage', reason: 'missing_fields' } satisfies SessionFailure;
        if (Date.parse(stored.expiresAt) <= Date.now()) throw { kind: 'expired' } satisfies SessionFailure;
        let renewable = true;
        if (Date.parse(stored.expiresAt) - Date.now() < sessionRefreshLeadMs) {
          const previous = stored.expiresAt; stored = await refreshSession(stored); renewable = sessionExpiryAdvanced(previous, stored.expiresAt);
        }
        await activate(stored, renewable);
      } catch (cause) {
        const raw = await SecureStore.getItemAsync(storageKey);
        if (raw) try { await handleSessionFailure(cause, JSON.parse(raw) as Session); }
        catch { await clearSession('errors.localSessionUnreadable', 'local_session_unreadable'); }
      }
      finally { setReady(true); }
    })();
  }, [discovery]);

  useEffect(() => {
    if (response?.type !== 'success' || !request?.codeVerifier || !discovery) return;
    (async () => {
      try {
        const exchanged = await AuthSession.exchangeCodeAsync({ clientId, code: response.params.code, redirectUri, extraParams: { code_verifier: request.codeVerifier! } }, discovery);
        if (!exchanged.idToken) throw new LocalizedError('errors.identityTokenMissing');
        let apiResponse: Response;
        try { apiResponse = await fetch(`${apiURL}/v1/sessions`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ identityToken: exchanged.idToken }) }); }
        catch { throw { kind: 'network' } satisfies SessionFailure; }
        const body = await apiResponse.json() as { data?: Session };
        if (!apiResponse.ok) throw sessionFailureFromResponse(apiResponse.status);
        if (!isValidSessionCredential(body.data)) throw sessionFailureFromResponse(502);
        await activate(body.data);
      } catch (cause) {
        if (session) await handleSessionFailure(cause, session);
        else { setAccessState('authentication_required'); setErrorKey(localizedFailure(cause, 'errors.signInFailed')); }
      }
      finally { setReady(true); }
    })();
  }, [response, request?.codeVerifier, discovery]);

  useEffect(() => {
    if (!session) return;
    const untilExpiry = Date.parse(session.expiresAt) - Date.now();
    if (untilExpiry <= 0) { void clearSession('errors.sessionExpired'); return; }
    const delay = accessState === 'authenticated_offline' && retryAttempt > 0 ? sessionRetryDelay(retryAttempt, session.expiresAt) : sessionRenewable ? sessionRefreshDelay(session.expiresAt) : untilExpiry;
    const timer = setTimeout(() => {
      if (!sessionRenewable && (accessState !== 'authenticated_offline' || retryAttempt === 0)) { void clearSession('errors.sessionExpired'); return; }
      void (async () => {
        let next: Session;
        try { next = await refreshSession(session); }
        catch (cause) { await handleSessionFailure(cause, session); return; }
        try { await activate(next, sessionExpiryAdvanced(session.expiresAt, next.expiresAt)); }
        catch (cause) { await handleSessionFailure(cause, next); }
      })();
    }, delay);
    return () => clearTimeout(timer);
  }, [session?.token, session?.expiresAt, sessionRenewable, accessState, retryAttempt]);

  return <SafeAreaView style={{ padding: 32, gap: 16 }}>
    <Text style={{ fontSize: 32 }}>{i18n.t('app.title')}</Text>
    <Text>{!ready ? i18n.t('common.loading') : accessState === 'authenticated_offline' ? i18n.t('auth.offline') : profile ? i18n.t('auth.signedInAs', { name: profile.displayName || profile.email }) : i18n.t('auth.ready')}</Text>
    {errorKey ? <Text accessibilityRole="alert">{i18n.t(errorKey)}</Text> : null}
    <Button title={i18n.t(session ? 'auth.signOut' : 'auth.signIn')} disabled={!ready || (!session && !request)} onPress={() => session ? void clearSession() : void prompt()} />
  </SafeAreaView>;
}
