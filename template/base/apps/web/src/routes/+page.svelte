<script lang="ts">
  import { onMount } from 'svelte';
  import { createApiClient, sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
  import { classifySessionFailure, isSessionFailure, retainedSessionExpiry, sessionRetryDelay, validateSessionCredential, type SessionAccessState, type SessionFailure } from '@__APP_SLUG__/client-core';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { applicationSession, clearApplicationSession, createUserManager, refreshApplicationSession, revokeApplicationSession, type ApplicationSession } from '$lib/auth';
  import { apiURL } from '$lib/config';

  export let data: { locale: SupportedLocale };
  let profile: { id: string; email: string; displayName: string } | null = null;
  let ready = false;
  let errorKey: MessageKey | null = null;
	let accessState: SessionAccessState = 'authentication_required';
	let examples: Array<{ id: string; name: string }> = [];
	let exampleName = '';
	let examplesLoading = false;
	let exampleCreated = false;
	let refreshTimer: ReturnType<typeof setTimeout> | undefined;
	let refreshAttempt = 0;
	let i18n: Translator;
	$: i18n = createTranslator([data.locale]);

	function scheduleExpiration(expiresAt: string) {
		if (refreshTimer) clearTimeout(refreshTimer);
		refreshTimer = setTimeout(() => {
			clearApplicationSession();
			profile = null;
			examples = [];
			errorKey = 'errors.sessionExpired';
		}, Math.max(0, Date.parse(expiresAt) - Date.now()));
	}

	function scheduleRetry(expiresAt: string) {
		const delay = sessionRetryDelay(refreshAttempt, expiresAt);
		refreshAttempt += 1;
		if (delay <= 0) { scheduleExpiration(expiresAt); return; }
		if (refreshTimer) clearTimeout(refreshTimer);
		refreshTimer = setTimeout(() => void attemptRefresh(expiresAt), delay);
	}

	async function loadProfile(session: ApplicationSession) {
		profile = await validateSessionCredential(session, async (current) =>
			fetch(`${apiURL}/v1/me`, { headers: { Authorization: `Bearer ${current.token}` } }),
		);
	}

	async function attemptRefresh(expiresAt: string) {
		try {
			const next = await refreshApplicationSession();
			await loadProfile(next);
			await loadExamples(next.token);
			accessState = 'authenticated_online';
			refreshAttempt = 0;
			if (sessionExpiryAdvanced(expiresAt, next.expiresAt)) scheduleRefresh(next.expiresAt);
			else scheduleExpiration(next.expiresAt);
		} catch (cause) {
			const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
			const decision = classifySessionFailure(failure);
			if (decision.discardCredential) { clearApplicationSession(); profile = null; examples = []; }
			else {
				const retainedExpiry = retainedSessionExpiry(applicationSession(), expiresAt);
				accessState = 'authenticated_offline';
				if (decision.retryable) scheduleRetry(retainedExpiry);
				else scheduleExpiration(retainedExpiry);
			}
			errorKey = decision.retryable ? 'errors.temporarilyUnavailable' : decision.discardCredential ? 'errors.sessionExpired' : 'errors.apiRejected';
		}
	}

	function scheduleRefresh(expiresAt: string) {
		if (refreshTimer) clearTimeout(refreshTimer);
		refreshTimer = setTimeout(() => void attemptRefresh(expiresAt), sessionRefreshDelay(expiresAt));
	}

	async function loadExamples(token: string) {
		examplesLoading = true;
		try {
			const client = createApiClient(apiURL, () => token);
			const result = await client.GET('/v1/examples', { params: { header: { Authorization: `Bearer ${token}` }, query: { limit: 50 } } });
			if (result.error || !result.data?.data) throw new Error();
			examples = result.data.data;
		} catch { errorKey = 'errors.examplesLoadFailed'; }
		finally { examplesLoading = false; }
	}

	async function createExample() {
		const session = applicationSession();
		const name = exampleName.trim();
		if (!session || !name) return;
		exampleCreated = false;
		try {
			const client = createApiClient(apiURL, () => session.token);
			const result = await client.POST('/v1/examples', { params: { header: { Authorization: `Bearer ${session.token}`, 'Idempotency-Key': crypto.randomUUID() } }, body: { name } });
			if (result.error || !result.data?.data) throw new Error();
			examples = [...examples, result.data.data];
			exampleName = '';
			exampleCreated = true;
		} catch { errorKey = 'errors.exampleCreateFailed'; }
	}

  onMount(async () => {
    try {
		let session = applicationSession();
		if (session) {
			if (Date.parse(session.expiresAt) - Date.now() < sessionRefreshLeadMs) {
				const previousExpiry = session.expiresAt;
				session = await refreshApplicationSession();
				if (sessionExpiryAdvanced(previousExpiry, session.expiresAt)) scheduleRefresh(session.expiresAt);
				else scheduleExpiration(session.expiresAt);
			} else scheduleRefresh(session.expiresAt);
			await loadProfile(session);
			accessState = 'authenticated_online';
		await loadExamples(session.token);
      }
    } catch (cause) {
	  const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
	  const decision = classifySessionFailure(failure);
	  if (decision.discardCredential) clearApplicationSession();
	  else {
		accessState = 'authenticated_offline';
		const current = applicationSession();
		if (current) decision.retryable ? scheduleRetry(current.expiresAt) : scheduleExpiration(current.expiresAt);
	  }
	  errorKey = decision.retryable ? 'errors.temporarilyUnavailable' : decision.discardCredential ? (decision.state === 'local_session_unreadable' ? 'errors.localSessionUnreadable' : 'errors.sessionExpired') : 'errors.apiRejected';
    } finally {
      ready = true;
    }
  });

  async function signIn() {
    errorKey = null;
    try { await createUserManager().signinRedirect(); }
    catch { errorKey = 'errors.signInFailed'; }
  }

  async function signOut() {
	if (refreshTimer) clearTimeout(refreshTimer);
    await revokeApplicationSession();
	accessState = 'authentication_required';
    profile = null;
	examples = [];
  }
</script>

<main>
  <h1>{i18n.t('app.title')}</h1>
  {#if !ready}<p>{i18n.t('common.loading')}</p>
	{:else if accessState === 'authenticated_offline' && applicationSession()}<p>{i18n.t('auth.offline')}</p><button onclick={signOut}>{i18n.t('auth.signOut')}</button>
	{:else if profile}<p>{i18n.t('auth.signedInAs', { name: profile.displayName || profile.email })}</p><button onclick={signOut}>{i18n.t('auth.signOut')}</button>
	<section>
		<h2>{i18n.t('examples.heading')}</h2>
		{#if examplesLoading}<p>{i18n.t('common.loading')}</p>
		{:else if examples.length === 0}<p>{i18n.t('examples.empty')}</p>
		{:else}<ul>{#each examples as example (example.id)}<li>{example.name}</li>{/each}</ul>{/if}
		<form onsubmit={(event) => { event.preventDefault(); void createExample(); }}>
			<label>{i18n.t('examples.name')}<input bind:value={exampleName} required minlength="1" /></label>
			<button disabled={!exampleName.trim()}>{i18n.t('examples.create')}</button>
		</form>
		{#if exampleCreated}<p role="status">{i18n.t('examples.created')}</p>{/if}
	</section>
  {:else}<p>{i18n.t('app.ready')}</p><button onclick={signIn}>{i18n.t('auth.signIn')}</button>{/if}
  {#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{/if}
</main>
<style>main{font:16px system-ui;max-width:42rem;margin:10vh auto;padding:2rem}section{margin-top:2rem}form,label{display:flex;gap:.75rem;align-items:center}input{padding:.6rem;border:1px solid #777;border-radius:.4rem}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#111;color:white}button:disabled{opacity:.5}</style>
