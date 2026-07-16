<script lang="ts">import { onMount } from 'svelte'; import { UserManager } from 'oidc-client-ts'; onMount(async()=>{const m=new UserManager({authority:import.meta.env.PUBLIC_OIDC_ISSUER,client_id:import.meta.env.PUBLIC_OIDC_CLIENT_ID,redirect_uri:`${location.origin}/callback`});await m.signinRedirectCallback();location.href='/'});</script>
<p>Signing you in…</p>

