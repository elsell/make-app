<script lang="ts">
  import { UserManager, WebStorageStateStore } from 'oidc-client-ts';
  import { createApiClient } from '@__APP_SLUG__/api-client';
  const issuer = import.meta.env.PUBLIC_OIDC_ISSUER ?? 'http://localhost:5556/dex';
  const manager = new UserManager({ authority: issuer, client_id: import.meta.env.PUBLIC_OIDC_CLIENT_ID ?? '__APP_SLUG__-web', redirect_uri: `${location.origin}/callback`, response_type: 'code', scope: 'openid profile email', userStore: new WebStorageStateStore({ store: localStorage }) });
  let profile: { id:string; email:string; displayName:string } | null = null;
  async function load() { const user=await manager.getUser(); if(!user)return; const client=createApiClient(import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080',()=>user.access_token); const result=await client.GET('/v1/me', { params: { header: { Authorization: `Bearer ${user.access_token}` } } }); profile=result.data?.data ?? null; }
  load();
</script>
<main><h1>__APP_NAME__</h1>{#if profile}<p>Signed in as {profile.displayName || profile.email}</p><button onclick={() => manager.signoutRedirect()}>Sign out</button>{:else}<p>Your generated application is ready.</p><button onclick={() => manager.signinRedirect()}>Sign in</button>{/if}</main>
<style>main{font:16px system-ui;max-width:42rem;margin:10vh auto;padding:2rem}button{padding:.65rem 1rem;border:0;border-radius:.5rem;background:#111;color:white}</style>
