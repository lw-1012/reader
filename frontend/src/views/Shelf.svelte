<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { go, showToast } from '../lib/stores.js';

  let books = [];
  let loading = true;
  let importing = false;
  let fileInput;

  async function load() {
    loading = true;
    try {
      books = await api.listBooks();
    } catch (e) {
      showToast(e.message);
    } finally {
      loading = false;
    }
  }
  onMount(load);

  async function onFile(e) {
    const file = e.target.files[0];
    if (!file) return;
    importing = true;
    try {
      const res = await api.importBook(file);
      showToast(`导入完成（${res.paragraphs} 段）`);
      await load();
    } catch (e) {
      showToast('导入失败: ' + e.message);
    } finally {
      importing = false;
      fileInput.value = '';
    }
  }

  async function del(b, ev) {
    ev.stopPropagation();
    if (!confirm(`删除《${b.title}》？此操作不可撤销。`)) return;
    try {
      await api.deleteBook(b.id);
      books = books.filter(x => x.id !== b.id);
    } catch (e) { showToast(e.message); }
  }

  function open(b) {
    const start = b.progress || 1;
    go(`/read/${b.id}?p=${start}`);
  }
</script>

<div class="app-shell">
  <div class="topbar">
    <span class="title">书架</span>
    <button class="icon ghost" on:click={() => fileInput.click()} title="导入 JSON">＋</button>
    <button class="icon ghost" on:click={() => go('/settings')} title="设置">⚙</button>
    <input bind:this={fileInput} type="file" accept="application/json,.json" on:change={onFile} style="display:none" />
  </div>
  <div class="content">
    <div class="page-pad">
      {#if importing}
        <div class="muted row"><span class="spinner" /> <span>正在导入…</span></div>
      {/if}
      {#if loading}
        <div class="muted">加载中…</div>
      {:else if books.length === 0}
        <div class="empty">
          <h2>书架空空如也</h2>
          <p class="muted">点击右上角 <strong>＋</strong> 导入一本结构化 JSON 书籍。</p>
        </div>
      {:else}
        <div class="grid">
          {#each books as b (b.id)}
            <article class="card" on:click={() => open(b)} on:keydown={(e) => e.key === 'Enter' && open(b)} role="button" tabindex="0">
              <div class="t">{b.title}</div>
              {#if b.author}<div class="a">{b.author}</div>{/if}
              <div class="meta">
                <span>{b.paragraphs} 段</span>
                {#if b.progress}
                  <span class="dot">·</span>
                  <span>已读到 {b.progress}（{Math.round((b.progress / b.paragraphs) * 100)}%）</span>
                {/if}
              </div>
              <button class="ghost del" on:click={(e) => del(b, e)} aria-label="删除">×</button>
            </article>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .grid { display: grid; grid-template-columns: 1fr; gap: 0.7rem; }
  @media (min-width: 640px) { .grid { grid-template-columns: 1fr 1fr; } }
  .card {
    position: relative;
    background: var(--bg-elev);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1rem 1.1rem;
    cursor: pointer;
    transition: transform 100ms ease, background 120ms ease;
  }
  .card:active { transform: scale(0.99); background: var(--bg-elev-2); }
  .t { font-size: 1.05rem; font-weight: 600; line-height: 1.3; padding-right: 2rem; }
  .a { color: var(--fg-dim); font-size: 0.85rem; margin-top: 0.15rem; }
  .meta { color: var(--fg-mute); font-size: 0.78rem; margin-top: 0.5rem; display: flex; gap: 0.4rem; align-items: center; }
  .dot { opacity: 0.5; }
  .del { position: absolute; top: 0.4rem; right: 0.4rem; color: var(--fg-mute); font-size: 1.4rem; line-height: 1; padding: 0.2rem 0.5rem; }
  .del:hover { color: var(--danger); }
  .empty { padding-top: 8vh; text-align: center; }
</style>
