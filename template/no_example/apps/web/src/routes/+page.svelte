<script lang="ts">
  import { onMount } from 'svelte';
  import { sessionExpiryAdvanced, sessionRefreshDelay, sessionRefreshLeadMs } from '@__APP_SLUG__/api-client';
  import { classifySessionFailure, isSessionFailure, retainedSessionExpiry, sessionRetryDelay, validateSessionCredential, type ClientRuntimeConfig, type SessionAccessState, type SessionFailure } from '@__APP_SLUG__/client-core';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { applicationSession, clearApplicationSession, createUserManager, refreshApplicationSession, revokeApplicationSession, type ApplicationSession } from '$lib/auth';

  export let data: { locale: SupportedLocale; config: ClientRuntimeConfig };
  let profile: { id: string; email: string; displayName: string } | null = null;
  let ready = false;
  let errorKey: MessageKey | null = null;
  let accessState: SessionAccessState = 'authentication_required';
  let refreshTimer: ReturnType<typeof setTimeout> | undefined;
  let refreshAttempt = 0;
  let i18n: Translator;
  $: i18n = createTranslator([data.locale]);

  function scheduleExpiration(expiresAt: string) {
    if (refreshTimer) clearTimeout(refreshTimer);
    refreshTimer = setTimeout(() => {
      clearApplicationSession();
      profile = null;
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
      fetch(`${data.config.apiURL}/v1/me`, { headers: { Authorization: `Bearer ${current.token}` } }),
    );
  }

  async function attemptRefresh(expiresAt: string) {
    try {
      const next = await refreshApplicationSession(data.config);
      await loadProfile(next);
      accessState = 'authenticated_online';
      refreshAttempt = 0;
      if (sessionExpiryAdvanced(expiresAt, next.expiresAt)) scheduleRefresh(next.expiresAt);
      else scheduleExpiration(next.expiresAt);
    } catch (cause) {
      const failure: SessionFailure = isSessionFailure(cause) ? cause : { kind: 'network' };
      const decision = classifySessionFailure(failure);
      if (decision.discardCredential) { clearApplicationSession(); profile = null; }
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

  onMount(async () => {
    try {
      let session = applicationSession();
      if (session) {
        if (Date.parse(session.expiresAt) - Date.now() < sessionRefreshLeadMs) {
          const previousExpiry = session.expiresAt;
          session = await refreshApplicationSession(data.config);
          if (sessionExpiryAdvanced(previousExpiry, session.expiresAt)) scheduleRefresh(session.expiresAt);
          else scheduleExpiration(session.expiresAt);
        } else scheduleRefresh(session.expiresAt);
        await loadProfile(session);
        accessState = 'authenticated_online';
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
    } finally { ready = true; }
  });

  async function signIn() {
    errorKey = null;
    try { await createUserManager(data.config).signinRedirect(); }
    catch { errorKey = 'errors.signInFailed'; }
  }

  async function signOut() {
    if (refreshTimer) clearTimeout(refreshTimer);
    await revokeApplicationSession(data.config);
    accessState = 'authentication_required';
    profile = null;
  }
</script>

<main>
  <h1>{i18n.t('app.title')}</h1>
  {#if !ready}<p>{i18n.t('common.loading')}</p>
  {:else if accessState === 'authenticated_offline' && applicationSession()}<p>{i18n.t('auth.offline')}</p><button onclick={signOut}>{i18n.t('auth.signOut')}</button>
  {:else if profile}<p>{i18n.t('auth.signedInAs', { name: profile.displayName || profile.email })}</p><button onclick={signOut}>{i18n.t('auth.signOut')}</button>
  {:else}<p>{i18n.t('app.ready')}</p><button onclick={signIn}>{i18n.t('auth.signIn')}</button>{/if}
  {#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{/if}
</main>
<style>main{font:16px system-ui;max-width:42rem;margin:10vh auto;padding:2rem}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#111;color:white}</style>
