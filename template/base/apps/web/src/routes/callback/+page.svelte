<script lang="ts">
  import { onMount } from 'svelte';
  import type { ClientRuntimeConfig } from '@__APP_SLUG__/client-core';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { createUserManager, exchangeApplicationSession } from '$lib/auth';

  export let data: { locale: SupportedLocale; config: ClientRuntimeConfig };
  let errorKey: MessageKey | null = null;
  let i18n: Translator;
  $: i18n = createTranslator([data.locale]);

  onMount(async () => {
    const manager = createUserManager(data.config);
    try {
      const user = await manager.signinRedirectCallback();
      if (!user.id_token) throw new Error('identity_token_missing');
      await exchangeApplicationSession(user.id_token, data.config);
      window.location.replace('/');
    }
    catch { errorKey = 'errors.callbackFailed'; }
    finally {
      await manager.removeUser().catch(() => undefined);
      await manager.clearStaleState().catch(() => undefined);
    }
  });
</script>

{#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{:else}<p>{i18n.t('auth.signingIn')}</p>{/if}
