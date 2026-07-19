import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as Crypto from 'expo-crypto';
import * as WebBrowser from 'expo-web-browser';
import { getLocales } from 'expo-localization';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text, TextInput, View } from 'react-native';
import { createApiClient, sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
import { classifySessionFailure, isSessionFailure, isValidSessionCredential, publicEndpointConfig, refreshSessionCredential, sessionFailureFromResponse, sessionRetryDelay, validateSessionCredential, type SessionAccessState, type SessionFailure } from '@__APP_SLUG__/client-core';
import { type MessageKey } from '@__APP_SLUG__/i18n';
import { createDeviceTranslator } from '../src/i18n';

WebBrowser.maybeCompleteAuthSession();
const productionBuild = process.env.EXPO_PUBLIC_APP_ENV === 'production';
function publicConfig(value: string | undefined, developmentDefault: string, name: string): string {
  if (value) return value;
  if (productionBuild) throw new Error(name);
  return developmentDefault;
}
const issuer = publicEndpointConfig(process.env.EXPO_PUBLIC_OIDC_ISSUER, 'http://localhost:5556/dex', 'EXPO_PUBLIC_OIDC_ISSUER', productionBuild);
const clientId = publicConfig(process.env.EXPO_PUBLIC_OIDC_CLIENT_ID, '__APP_SLUG__-mobile', 'EXPO_PUBLIC_OIDC_CLIENT_ID');
const apiURL = publicEndpointConfig(process.env.EXPO_PUBLIC_API_URL, 'http://localhost:8080', 'EXPO_PUBLIC_API_URL', productionBuild);
const storageKey = 'application_session';
const i18n = createDeviceTranslator(getLocales);

type Session = { token: string; expiresAt: string };
type Profile = { id: string; email: string; displayName: string };

class LocalizedError extends Error {
  constructor(readonly key: MessageKey) { super(key); }
}

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

function localizedFailure(cause: unknown, fallback: MessageKey): MessageKey {
  return cause instanceof LocalizedError ? cause.key : fallback;
}

export default function Home() {
  const [session, setSession] = useState<Session | null>(null);
	const [accessState, setAccessState] = useState<SessionAccessState>('authentication_required');
  const [sessionRenewable, setSessionRenewable] = useState(true);
  const [retryAttempt, setRetryAttempt] = useState(0);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [ready, setReady] = useState(false);
  const [errorKey, setErrorKey] = useState<MessageKey | null>(null);
	const [examples, setExamples] = useState<Array<{ id: string; name: string }>>([]);
	const [exampleName, setExampleName] = useState('');
  const discovery = AuthSession.useAutoDiscovery(issuer);
  const redirectUri = AuthSession.makeRedirectUri({ scheme: '__APP_NATIVE_ID__', path: 'callback' });
  const [request, response, prompt] = AuthSession.useAuthRequest({ clientId, redirectUri, scopes: ['openid', 'profile', 'email'], usePKCE: true }, discovery);

  async function activate(next: Session, renewable = true) {
    await saveSession(next);
    setSessionRenewable(renewable);
    setSession(next);
	const nextProfile = await validateSessionCredential<Profile>(next, async (current) =>
		fetch(`${apiURL}/v1/me`, { headers: { Authorization: `Bearer ${current.token}` } }),
	);
	setProfile(nextProfile);
	setAccessState('authenticated_online');
	setRetryAttempt(0);
	try {
		const resources = await createApiClient(apiURL, () => next.token).GET('/v1/examples', { params: { header: { Authorization: `Bearer ${next.token}` }, query: { limit: 50 } } });
		if (resources.error || !resources.data?.data) setErrorKey('errors.examplesLoadFailed');
		else setExamples(resources.data.data);
	} catch { setErrorKey('errors.examplesLoadFailed'); }
  }

	async function clearSession(message: MessageKey | null = null, state: SessionAccessState = 'authentication_required') {
    const token = session?.token;
    await saveSession(null);
    setSession(null);
	setRetryAttempt(0);
	setAccessState(state);
    setProfile(null);
	setExamples([]);
    setErrorKey(message);
    if (token) {
      try { await fetch(`${apiURL}/v1/session`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } }); }
      catch { /* Secure local disposal does not depend on the API being reachable. */ }
    }
  }

	async function handleSessionFailure(cause: unknown, current: Session) {
		const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
		const decision = classifySessionFailure(failure);
		if (decision.discardCredential) {
			const message: MessageKey = decision.state === 'local_session_unreadable' ? 'errors.localSessionUnreadable' : 'errors.sessionExpired';
			await clearSession(message, decision.state);
			return;
		}
		await saveSession(current);
		setSession(current);
		if (!decision.retryable) { setSessionRenewable(false); setRetryAttempt(0); }
		setAccessState('authenticated_offline');
		if (decision.retryable) setRetryAttempt((attempt) => attempt + 1);
		setErrorKey(decision.retryable ? 'errors.temporarilyUnavailable' : 'errors.apiRejected');
	}

  useEffect(() => {
    if (!discovery) return;
    (async () => {
      try {
        const raw = await SecureStore.getItemAsync(storageKey);
        if (!raw) return;
		let stored: Session;
		try { stored = JSON.parse(raw) as Session; }
		catch { throw { kind: 'local_storage', reason: 'malformed' } satisfies SessionFailure; }
		if (!stored.token || !stored.expiresAt || !Number.isFinite(Date.parse(stored.expiresAt))) throw { kind: 'local_storage', reason: 'missing_fields' } satisfies SessionFailure;
		if (Date.parse(stored.expiresAt) <= Date.now()) throw { kind: 'expired' } satisfies SessionFailure;
		let renewable = true;
		if (Date.parse(stored.expiresAt) - Date.now() < sessionRefreshLeadMs) {
			const previousExpiry = stored.expiresAt;
			stored = await refreshSession(stored);
			renewable = sessionExpiryAdvanced(previousExpiry, stored.expiresAt);
		}
		await activate(stored, renewable);
      } catch (cause) {
		const raw = await SecureStore.getItemAsync(storageKey);
		if (raw) {
			try { await handleSessionFailure(cause, JSON.parse(raw) as Session); }
			catch { await clearSession('errors.localSessionUnreadable', 'local_session_unreadable'); }
		}
      } finally { setReady(true); }
    })();
  }, [discovery]);

  useEffect(() => {
    if (response?.type !== 'success' || !request?.codeVerifier || !discovery) return;
    (async () => {
      try {
        const exchanged = await AuthSession.exchangeCodeAsync({ clientId, code: response.params.code, redirectUri, extraParams: { code_verifier: request.codeVerifier! } }, discovery);
        if (!exchanged.idToken) throw new LocalizedError('errors.identityTokenMissing');
        const headers = new Headers(); headers.set('Content-Type', 'application/json');
		let apiResponse: Response;
		try { apiResponse = await fetch(`${apiURL}/v1/sessions`, { method: 'POST', headers, body: JSON.stringify({ identityToken: exchanged.idToken }) }); }
		catch { throw { kind: 'network' } satisfies SessionFailure; }
		const body = await apiResponse.json() as { data?: Session };
		if (!apiResponse.ok) throw sessionFailureFromResponse(apiResponse.status);
		if (!isValidSessionCredential(body.data)) throw sessionFailureFromResponse(502);
        await activate(body.data);
      } catch (cause) {
		const current = session;
		if (current) await handleSessionFailure(cause, current);
		else { setAccessState('authentication_required'); setErrorKey(localizedFailure(cause, 'errors.signInFailed')); }
      } finally { setReady(true); }
    })();
  }, [response, request?.codeVerifier, discovery]);

	useEffect(() => {
		if (!session) return;
		const untilExpiry = Date.parse(session.expiresAt) - Date.now();
		if (untilExpiry <= 0) { void clearSession('errors.sessionExpired'); return; }
		const delay = accessState === 'authenticated_offline' && retryAttempt > 0
			? sessionRetryDelay(retryAttempt, session.expiresAt)
			: sessionRenewable
			? sessionRefreshDelay(session.expiresAt)
			: untilExpiry;
		const timer = setTimeout(() => {
			if (!sessionRenewable && (accessState !== 'authenticated_offline' || retryAttempt === 0)) {
				void clearSession('errors.sessionExpired');
				return;
			}
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

	async function createExample() {
		const name = exampleName.trim();
		if (!session || !name) return;
		try {
			const result = await createApiClient(apiURL, () => session.token).POST('/v1/examples', { params: { header: { Authorization: `Bearer ${session.token}`, 'Idempotency-Key': Crypto.randomUUID() } }, body: { name } });
			if (result.error || !result.data?.data) throw new LocalizedError('errors.exampleCreateFailed');
			setExamples((current) => [...current, result.data!.data!]);
			setExampleName('');
			setErrorKey(null);
		} catch (cause) { setErrorKey(localizedFailure(cause, 'errors.exampleCreateFailed')); }
	}

  return <SafeAreaView style={{ padding: 32, gap: 16 }}>
    <Text style={{ fontSize: 32 }}>{i18n.t('app.title')}</Text>
	<Text>{!ready ? i18n.t('common.loading') : accessState === 'authenticated_offline' ? i18n.t('auth.offline') : profile ? i18n.t('auth.signedInAs', { name: profile.displayName || profile.email }) : i18n.t('auth.ready')}</Text>
    {errorKey ? <Text accessibilityRole="alert">{i18n.t(errorKey)}</Text> : null}
    <Button title={i18n.t(session ? 'auth.signOut' : 'auth.signIn')} disabled={!ready || (!session && !request)} onPress={() => session ? void clearSession() : void prompt()} />
	{session ? <View style={{ gap: 12 }}>
		<Text style={{ fontSize: 24 }}>{i18n.t('examples.heading')}</Text>
		<Text>{examples.length ? i18n.t('examples.count', { count: examples.length }) : i18n.t('examples.empty')}</Text>
		{examples.map((example) => <Text key={example.id}>{example.name}</Text>)}
		<TextInput accessibilityLabel={i18n.t('examples.name')} placeholder={i18n.t('examples.name')} value={exampleName} onChangeText={setExampleName} style={{ borderWidth: 1, padding: 12 }} />
		<Button title={i18n.t('examples.create')} disabled={!exampleName.trim()} onPress={() => void createExample()} />
	</View> : null}
  </SafeAreaView>;
}
