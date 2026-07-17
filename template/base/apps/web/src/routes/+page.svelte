<script lang="ts">
  import { onMount } from 'svelte';
  import { createApiClient } from '@__APP_SLUG__/api-client';
  import { createUserManager } from '$lib/auth';

  let profile: { id: string; email: string; displayName: string } | null = null;
  let ready = false;
  let error = '';

  onMount(async () => {
    const manager = createUserManager();
    manager.events.addAccessTokenExpired(async () => {
      await manager.removeUser();
      profile = null;
      error = 'Your session expired. Sign in again.';
    });
    try {
      const user = await manager.getUser();
      if (user && !user.expired) {
        const client = createApiClient(import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080', () => user.access_token);
        const result = await client.GET('/v1/me', { params: { header: { Authorization: `Bearer ${user.access_token}` } } });
        if (result.error) throw new Error('The API rejected this session.');
        profile = result.data?.data ?? null;
      } else if (user) {
        await manager.removeUser();
      }
    } catch (cause) {
      await manager.removeUser();
      error = cause instanceof Error ? cause.message : 'Sign-in failed.';
    } finally {
      ready = true;
    }
  });

  async function signIn() { error = ''; await createUserManager().signinRedirect(); }
  async function signOut() { const manager = createUserManager(); await manager.removeUser(); profile = null; await manager.signoutRedirect(); }
</script>

<main>
  <h1>__APP_NAME__</h1>
  {#if !ready}<p>Loading…</p>
  {:else if profile}<p>Signed in as {profile.displayName || profile.email}</p><button onclick={signOut}>Sign out</button>
  {:else}<p>Your generated application is ready.</p><button onclick={signIn}>Sign in</button>{/if}
  {#if error}<p role="alert">{error}</p>{/if}
</main>
<style>main{font:16px system-ui;max-width:42rem;margin:10vh auto;padding:2rem}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#111;color:white}</style>
