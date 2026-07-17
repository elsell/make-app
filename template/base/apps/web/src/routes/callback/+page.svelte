<script lang="ts">
  import { onMount } from 'svelte';
  import { createUserManager } from '$lib/auth';
  let error = '';
  onMount(async () => {
    try { await createUserManager().signinRedirectCallback(); window.location.replace('/'); }
    catch (cause) { error = cause instanceof Error ? cause.message : 'Sign-in callback failed.'; }
  });
</script>

{#if error}<p role="alert">{error}</p>{:else}<p>Signing you in…</p>{/if}
