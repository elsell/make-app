<script lang="ts">
  import { onMount } from 'svelte';
  import { createApiClient } from '@__APP_SLUG__/api-client';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { createUserManager } from '$lib/auth';

  export let data: { locale: SupportedLocale };
  let profile: { id: string; email: string; displayName: string } | null = null;
  let ready = false;
  let errorKey: MessageKey | null = null;
  let i18n: Translator;
  $: i18n = createTranslator([data.locale]);

  onMount(async () => {
    const manager = createUserManager();
    manager.events.addAccessTokenExpired(async () => {
      await manager.removeUser();
      profile = null;
      errorKey = 'errors.sessionExpired';
    });
    try {
      const user = await manager.getUser();
      if (user && !user.expired) {
        if (!user.id_token) {
          errorKey = 'errors.identityTokenMissing';
          await manager.removeUser();
          return;
        }
        const client = createApiClient(import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080', () => user.id_token!);
        const result = await client.GET('/v1/me', { params: { header: { Authorization: `Bearer ${user.id_token}` } } });
        if (result.error) {
          errorKey = 'errors.apiRejected';
          await manager.removeUser();
          return;
        }
        profile = result.data?.data ?? null;
      } else if (user) {
        await manager.removeUser();
      }
    } catch {
      await manager.removeUser();
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
    const manager = createUserManager();
    await manager.removeUser();
    profile = null;
    await manager.signoutRedirect();
  }
</script>

<main>
  <h1>{i18n.t('app.title')}</h1>
  {#if !ready}<p>{i18n.t('common.loading')}</p>
  {:else if profile}<p>{i18n.t('auth.signedInAs', { name: profile.displayName || profile.email })}</p><button onclick={signOut}>{i18n.t('auth.signOut')}</button>
  {:else}<p>{i18n.t('app.ready')}</p><button onclick={signIn}>{i18n.t('auth.signIn')}</button>{/if}
  {#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{/if}
</main>
<style>main{font:16px system-ui;max-width:42rem;margin:10vh auto;padding:2rem}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#111;color:white}</style>
