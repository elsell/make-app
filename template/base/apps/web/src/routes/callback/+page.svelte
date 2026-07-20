<script lang="ts">
  import { onMount } from 'svelte';
  import type { ClientRuntimeConfig } from '@__APP_SLUG__/client-core';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { completeApplicationSignIn } from '$lib/auth';

  export let data: { locale: SupportedLocale; config: ClientRuntimeConfig };
  let errorKey: MessageKey | null = null;
  let i18n: Translator;
  $: i18n = createTranslator([data.locale]);

  onMount(async () => {
    try {
      await completeApplicationSignIn(data.config);
      window.location.replace('/');
    }
    catch { errorKey = 'errors.callbackFailed'; }
  });
</script>

{#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{:else}<p>{i18n.t('auth.signingIn')}</p>{/if}
