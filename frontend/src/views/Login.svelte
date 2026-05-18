<script>
  import { createEventDispatcher } from 'svelte';
  import { api } from '../lib/api.js';
  import { showToast } from '../lib/stores.js';

  const dispatch = createEventDispatcher();
  let password = '';
  let busy = false;

  async function submit() {
    if (!password) return;
    busy = true;
    try {
      await api.login(password);
      dispatch('authed');
    } catch (e) {
      showToast(e.message || '登录失败');
    } finally {
      busy = false;
    }
  }
</script>

<div class="app-shell">
  <div class="content">
    <div class="page-pad center">
      <h1>Reader</h1>
      <p class="muted small">单用户阅读器，输入密码以继续</p>
      <form on:submit|preventDefault={submit}>
        <div class="field">
          <input type="password" bind:value={password} placeholder="密码" autofocus />
        </div>
        <button class="primary" type="submit" disabled={busy || !password} style="width:100%">
          {busy ? '登录中…' : '登录'}
        </button>
      </form>
      <p class="muted small mt">
        首次使用：通过环境变量 <code>READER_PASSWORD</code> 设置密码，否则默认是 <code>reader</code>。
      </p>
    </div>
  </div>
</div>

<style>
  .center { max-width: 360px; padding-top: 12vh; }
  h1 { margin: 0 0 0.25rem 0; font-size: 2rem; letter-spacing: -0.02em; }
  .mt { margin-top: 1rem; }
  code { background: var(--bg-elev-2); padding: 0.1rem 0.35rem; border-radius: 4px; font-size: 0.85em; }
</style>
