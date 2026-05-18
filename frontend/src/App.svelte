<script>
  import { onMount } from 'svelte';
  import { route, toast } from './lib/stores.js';
  import { api } from './lib/api.js';
  import Login from './views/Login.svelte';
  import Shelf from './views/Shelf.svelte';
  import Reader from './views/Reader.svelte';
  import Settings from './views/Settings.svelte';

  let authed = null;

  onMount(async () => {
    try {
      const r = await api.check();
      authed = r.authed;
    } catch {
      authed = false;
    }
    if (!authed && location.hash !== '#/login') {
      location.hash = '#/login';
    } else if (authed && (location.hash === '' || location.hash === '#/' || location.hash === '#/login')) {
      location.hash = '#/shelf';
    }
  });
</script>

{#if authed === null}
  <div class="boot">
    <span class="spinner" />
  </div>
{:else if $route.parts[0] === 'login'}
  <Login on:authed={() => { authed = true; location.hash = '#/shelf'; }} />
{:else if $route.parts[0] === 'shelf' || $route.parts.length === 0}
  <Shelf />
{:else if $route.parts[0] === 'read'}
  <Reader bookId={Number($route.parts[1])} startGlobal={Number($route.query.get('p') || 0)} />
{:else if $route.parts[0] === 'settings'}
  <Settings />
{:else}
  <div class="page-pad">未知页面 <a href="#/shelf">返回</a></div>
{/if}

{#if $toast}
  <div class="toast">{$toast}</div>
{/if}

<style>
  .boot { display: grid; place-items: center; height: 100%; }
</style>
