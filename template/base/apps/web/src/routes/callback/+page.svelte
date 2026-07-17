<script lang="ts">
  import { onMount } from 'svelte';
  import { createTranslator, type MessageKey, type SupportedLocale, type Translator } from '@__APP_SLUG__/i18n';
  import { createUserManager } from '$lib/auth';

  export let data: { locale: SupportedLocale };
  let errorKey: MessageKey | null = null;
  let i18n: Translator;
  $: i18n = createTranslator([data.locale]);

  onMount(async () => {
    try { await createUserManager().signinRedirectCallback(); window.location.replace('/'); }
    catch { errorKey = 'errors.callbackFailed'; }
  });
</script>

{#if errorKey}<p role="alert">{i18n.t(errorKey)}</p>{:else}<p>{i18n.t('auth.signingIn')}</p>{/if}
