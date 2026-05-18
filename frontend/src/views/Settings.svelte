<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { go, showToast } from '../lib/stores.js';
  import { playTTS } from '../lib/audio.js';

  let s = null;
  let apiKeyInput = '';
  let saving = false;

  onMount(async () => {
    try { s = await api.getSettings(); } catch (e) { showToast(e.message); }
  });

  async function save() {
    saving = true;
    try {
      const patch = { ...s };
      delete patch.api_key_set;
      delete patch.api_key_masked;
      if (apiKeyInput) patch.api_key = apiKeyInput;
      s = await api.saveSettings(patch);
      apiKeyInput = '';
      showToast('已保存');
    } catch (e) {
      showToast('保存失败: ' + e.message);
    } finally { saving = false; }
  }

  async function logout() {
    await api.logout();
    location.hash = '#/login';
    location.reload();
  }

  async function testVoice() {
    try {
      await playTTS('Hello, this is a quick test of the selected voice.', s.voice);
    } catch (e) { showToast(e.message); }
  }
</script>

<div class="app-shell">
  <div class="topbar">
    <button class="ghost icon" on:click={() => go('/shelf')} aria-label="返回">‹</button>
    <span class="title">设置</span>
    <button class="primary" on:click={save} disabled={saving || !s}>{saving ? '保存中' : '保存'}</button>
  </div>

  <div class="content">
    <div class="page-pad">
      {#if !s}
        <div class="muted">加载中…</div>
      {:else}
        <h3>OpenRouter</h3>
        <div class="field">
          <label>API Key {s.api_key_set ? '（已配置）' : '（未配置）'}</label>
          <input type="password" bind:value={apiKeyInput} placeholder={s.api_key_set ? '保留留空，输入新值则覆盖' : '粘贴你的 sk-or-...'} />
        </div>
        <div class="field">
          <label>Base URL</label>
          <input bind:value={s.base_url} />
        </div>

        <h3>模型</h3>
        <div class="field">
          <label>简化模型</label>
          <input bind:value={s.simplify_model} placeholder="openai/gpt-4o-mini" />
        </div>
        <div class="field">
          <label>逐句分析模型</label>
          <input bind:value={s.analyze_model} placeholder="openai/gpt-4o-mini" />
        </div>
        <div class="field">
          <label>TTS 模型</label>
          <input bind:value={s.tts_model} placeholder="openai/gpt-4o-mini-tts" />
        </div>
        <div class="field">
          <label>声音</label>
          <div class="row">
            <input bind:value={s.voice} placeholder="alloy / nova / shimmer / ..." />
            <button on:click={testVoice}>试听</button>
          </div>
        </div>

        <h3>默认难度</h3>
        <div class="field">
          <label>CEFR Level</label>
          <select bind:value={s.level}>
            <option>A1</option><option>A2</option><option>B1</option><option>B2</option><option>C1</option><option>C2</option>
          </select>
        </div>

        <h3>提示词</h3>
        <p class="muted small">支持占位符 <code>{'{LEVEL}'}</code> 和 <code>{'{TEXT}'}</code>。修改提示词后已有缓存自动失效。</p>
        <div class="field">
          <label>简化 Prompt</label>
          <textarea bind:value={s.simplify_prompt}></textarea>
        </div>
        <div class="field">
          <label>逐句分析 Prompt</label>
          <textarea bind:value={s.analyze_prompt}></textarea>
        </div>
        <div class="field">
          <label>TTS 朗读指令</label>
          <textarea bind:value={s.tts_instruction} style="min-height: 80px"></textarea>
        </div>

        <h3>会话</h3>
        <button on:click={logout}>退出登录</button>
      {/if}
    </div>
  </div>
</div>

<style>
  h3 { margin-top: 1.5rem; margin-bottom: 0.6rem; font-size: 0.95rem; color: var(--fg-dim); text-transform: uppercase; letter-spacing: 0.05em; }
  code { background: var(--bg-elev-2); padding: 0.05rem 0.3rem; border-radius: 4px; font-size: 0.85em; }
</style>
