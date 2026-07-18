<script lang="ts">
  import { onMount } from 'svelte';
  import { env } from '$env/dynamic/public';
  import { createApiClient, sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { applicationSession, clearApplicationSession, createUserManager, refreshApplicationSession, revokeApplicationSession } from '$lib/auth';

  export let data: { locale: SupportedLocale };
  let profile: { id: string; email: string; displayName: string } | null = null;
  let ready = false;
  let errorKey: MessageKey | null = null;
	let examples: Array<{ id: string; name: string }> = [];
	let exampleName = '';
	let examplesLoading = false;
	let exampleCreated = false;
	let refreshTimer: ReturnType<typeof setTimeout> | undefined;
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

	function scheduleRefresh(expiresAt: string) {
		if (refreshTimer) clearTimeout(refreshTimer);
		const delay = sessionRefreshDelay(expiresAt);
		refreshTimer = setTimeout(async () => {
			try {
				const next = await refreshApplicationSession();
				if (sessionExpiryAdvanced(expiresAt, next.expiresAt)) scheduleRefresh(next.expiresAt);
				else scheduleExpiration(next.expiresAt);
			}
	catch { clearApplicationSession(); profile = null; examples = []; errorKey = 'errors.sessionExpired'; }
		}, delay);
	}

	async function loadExamples(token: string) {
		examplesLoading = true;
		try {
			const client = createApiClient(env.PUBLIC_API_URL ?? 'http://localhost:8080', () => token);
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
			const client = createApiClient(env.PUBLIC_API_URL ?? 'http://localhost:8080', () => session.token);
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
			const client = createApiClient(env.PUBLIC_API_URL ?? 'http://localhost:8080', () => session?.token ?? '');
			const result = await client.GET('/v1/me', { params: { header: { Authorization: `Bearer ${session.token}` } } });
        if (result.error) {
          errorKey = 'errors.apiRejected';
          clearApplicationSession();
          return;
        }
        profile = result.data?.data ?? null;
		await loadExamples(session.token);
      }
    } catch {
      clearApplicationSession();
      errorKey = 'errors.signInFailed';
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
    profile = null;
	examples = [];
  }
</script>

<main>
  <h1>{i18n.t('app.title')}</h1>
  {#if !ready}<p>{i18n.t('common.loading')}</p>
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
