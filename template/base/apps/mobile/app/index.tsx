import * as AuthSession from 'expo-auth-session';
import * as SecureStore from 'expo-secure-store';
import * as Crypto from 'expo-crypto';
import * as WebBrowser from 'expo-web-browser';
import { getLocales } from 'expo-localization';
import { useEffect, useState } from 'react';
import { Button, SafeAreaView, Text, TextInput, View } from 'react-native';
import { createApiClient, sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
import { type MessageKey } from '@__APP_SLUG__/i18n';
import { createDeviceTranslator } from '../src/i18n';

WebBrowser.maybeCompleteAuthSession();
const issuer = process.env.EXPO_PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
const clientId = process.env.EXPO_PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-mobile';
const apiURL = process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080';
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
  const response = await fetch(`${apiURL}/v1/session/refresh`, { method: 'POST', headers: { Authorization: `Bearer ${session.token}` } });
  const body = await response.json() as { data?: Session };
  if (!response.ok || !body.data?.token || !body.data.expiresAt) throw new LocalizedError('errors.sessionExpired');
  return body.data;
}

function localizedFailure(cause: unknown, fallback: MessageKey): MessageKey {
  return cause instanceof LocalizedError ? cause.key : fallback;
}

export default function Home() {
  const [session, setSession] = useState<Session | null>(null);
  const [sessionRenewable, setSessionRenewable] = useState(true);
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
    const result = await createApiClient(apiURL, () => next.token).GET('/v1/me', { params: { header: { Authorization: `Bearer ${next.token}` } } });
    if (result.error || !result.data?.data) throw new LocalizedError('errors.apiRejected');
    setProfile(result.data.data);
	const resources = await createApiClient(apiURL, () => next.token).GET('/v1/examples', { params: { header: { Authorization: `Bearer ${next.token}` }, query: { limit: 50 } } });
	if (resources.error || !resources.data?.data) throw new LocalizedError('errors.examplesLoadFailed');
	setExamples(resources.data.data);
  }

  async function clearSession(message: MessageKey | null = null) {
    const token = session?.token;
    await saveSession(null);
    setSession(null);
    setProfile(null);
	setExamples([]);
    setErrorKey(message);
    if (token) {
      try { await fetch(`${apiURL}/v1/session`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } }); }
      catch { /* Secure local disposal does not depend on the API being reachable. */ }
    }
  }

  useEffect(() => {
    if (!discovery) return;
    (async () => {
      try {
        const raw = await SecureStore.getItemAsync(storageKey);
        if (!raw) return;
		let stored = JSON.parse(raw) as Session;
		if (!stored.token || Date.parse(stored.expiresAt) <= Date.now()) throw new LocalizedError('errors.sessionExpired');
		let renewable = true;
		if (Date.parse(stored.expiresAt) - Date.now() < sessionRefreshLeadMs) {
			const previousExpiry = stored.expiresAt;
			stored = await refreshSession(stored);
			renewable = sessionExpiryAdvanced(previousExpiry, stored.expiresAt);
		}
		await activate(stored, renewable);
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
        const headers = new Headers(); headers.set('Content-Type', 'application/json');
		const apiResponse = await fetch(`${apiURL}/v1/sessions`, { method: 'POST', headers, body: JSON.stringify({ identityToken: exchanged.idToken }) });
		const body = await apiResponse.json() as { data?: Session };
		if (!apiResponse.ok || !body.data?.token) throw new LocalizedError('errors.apiRejected');
        await activate(body.data);
      } catch (cause) {
        await clearSession(localizedFailure(cause, 'errors.signInFailed'));
      } finally { setReady(true); }
    })();
  }, [response, request?.codeVerifier, discovery]);

	useEffect(() => {
		if (!session) return;
		const delay = sessionRenewable
			? sessionRefreshDelay(session.expiresAt)
			: Math.max(0, Date.parse(session.expiresAt) - Date.now());
		const timer = setTimeout(() => {
			if (!sessionRenewable) {
				void clearSession('errors.sessionExpired');
				return;
			}
			void refreshSession(session)
				.then((next) => activate(next, sessionExpiryAdvanced(session.expiresAt, next.expiresAt)))
				.catch((cause) => clearSession(localizedFailure(cause, 'errors.sessionExpired')));
		}, delay);
		return () => clearTimeout(timer);
  }, [session?.token, session?.expiresAt, sessionRenewable]);

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
    <Text>{!ready ? i18n.t('common.loading') : profile ? i18n.t('auth.signedInAs', { name: profile.displayName || profile.email }) : i18n.t('auth.ready')}</Text>
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
