<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { api } from '../lib/api.js';
  import { go, showToast } from '../lib/stores.js';
  import { playTTS, stopTTS } from '../lib/audio.js';
  import { swipe } from '../lib/swipe.js';
  import TextSpans from '../components/TextSpans.svelte';
  import SentenceList from '../components/SentenceList.svelte';
  import Toc from '../components/Toc.svelte';

  export let bookId;
  export let startGlobal = 1;

  let book = null;
  let total = 0;
  let level = 'B1';
  let mode = 'original';   // 'original' | 'simplified'
  let cur = startGlobal || 1;
  let paragraph = null;
  let simpl = null;        // { simplified, sentences } or null
  let loadingPara = false;
  let loadingSim = false;
  let analysisCache = {};  // sentence text => analysis result
  let openSentenceIdx = null;
  let analyzingIdx = null;
  let tocOpen = false;
  let slideDir = 0; // for animation

  // touch hint
  let savedTimer;

  onMount(async () => {
    try {
      const [b, prog, settings] = await Promise.all([
        api.getBook(bookId),
        api.getProgress(bookId).catch(() => null),
        api.getSettings().catch(() => null)
      ]);
      book = b;
      total = b.paragraphs;
      if (settings) level = (prog && prog.level) || settings.level || 'B1';
      if (!startGlobal && prog && prog.last_global_index) cur = prog.last_global_index;
      cur = clamp(cur, 1, total);
      await loadParagraph(cur);
    } catch (e) {
      showToast(e.message);
    }
  });

  onDestroy(() => { stopTTS(); clearTimeout(savedTimer); });

  function clamp(n, lo, hi) { return Math.max(lo, Math.min(hi, n)); }

  async function loadParagraph(n) {
    loadingPara = true;
    openSentenceIdx = null;
    simpl = null;
    try {
      const list = await api.getParagraphs(bookId, n, n);
      paragraph = list[0] || null;
      // try cached simplification immediately (peek without force)
      if (paragraph && mode === 'simplified') await fetchSimplified(false);
    } catch (e) {
      showToast(e.message);
    } finally {
      loadingPara = false;
    }
    saveProgress();
    history.replaceState(null, '', `#/read/${bookId}?p=${cur}`);
    prefetch(n + 1);
  }

  async function prefetch(n) {
    if (n < 1 || n > total) return;
    try { await api.getParagraphs(bookId, n, n); } catch {}
  }

  async function fetchSimplified(forceRegen = false) {
    if (!paragraph) return;
    loadingSim = true;
    try {
      const r = await api.simplify(paragraph.id, level, forceRegen);
      simpl = { simplified: r.simplified, sentences: r.sentences || [] };
    } catch (e) {
      simpl = null;
      showToast('简化失败: ' + e.message);
    } finally {
      loadingSim = false;
    }
  }

  async function ensureSimplified() {
    if (mode !== 'simplified') return;
    if (!simpl) await fetchSimplified(false);
  }

  async function toggleMode() {
    mode = mode === 'original' ? 'simplified' : 'original';
    await ensureSimplified();
  }

  async function nav(delta) {
    const next = clamp(cur + delta, 1, total);
    if (next === cur) return;
    slideDir = delta;
    cur = next;
    await tick();
    await loadParagraph(cur);
    setTimeout(() => { slideDir = 0; }, 240);
  }

  async function changeLevel(newLevel) {
    if (newLevel === level) return;
    level = newLevel;
    if (mode === 'simplified') {
      simpl = null;
      await fetchSimplified(false);
    }
    saveProgress();
  }

  function saveProgress() {
    if (!paragraph) return;
    clearTimeout(savedTimer);
    savedTimer = setTimeout(() => {
      api.putProgress(bookId, paragraph.id, level).catch(() => {});
    }, 400);
  }

  async function openAnalysis(idx) {
    if (!simpl) return;
    const text = simpl.sentences[idx];
    if (!text) return;
    if (openSentenceIdx === idx) { openSentenceIdx = null; return; }
    openSentenceIdx = idx;
    if (analysisCache[text]) return;
    analyzingIdx = idx;
    try {
      analysisCache[text] = await api.analyze(text, level);
      analysisCache = { ...analysisCache };
    } catch (e) {
      showToast('分析失败: ' + e.message);
      openSentenceIdx = null;
    } finally {
      analyzingIdx = null;
    }
  }

  async function regenerate() {
    if (!paragraph) return;
    if (!confirm(`重新生成 ${level} 简化版本？将覆盖缓存。`)) return;
    simpl = null;
    await fetchSimplified(true);
  }

  function jumpTo(globalIdx) {
    tocOpen = false;
    if (globalIdx === cur) return;
    slideDir = globalIdx > cur ? 1 : -1;
    cur = globalIdx;
    loadParagraph(cur);
    setTimeout(() => { slideDir = 0; }, 240);
  }

  function onWord(e) { playTTS(e.detail); }
</script>

<div class="app-shell">
  <div class="topbar">
    <button class="ghost icon" on:click={() => go('/shelf')} aria-label="返回">‹</button>
    <div class="title">
      <div class="t">{book ? book.title : '加载中…'}</div>
      {#if paragraph}<div class="sub">{paragraph.section_title} · {cur}/{total}</div>{/if}
    </div>
    <button class="ghost icon" on:click={() => tocOpen = true} aria-label="目录">☰</button>
    <button class="ghost icon" on:click={() => go('/settings')} aria-label="设置">⚙</button>
  </div>

  <div class="modebar">
    <div class="tabs">
      <button class:active={mode === 'original'} on:click={() => { mode = 'original'; }}>原文</button>
      <button class:active={mode === 'simplified'} on:click={() => { mode = 'simplified'; ensureSimplified(); }}>简化 {level}</button>
    </div>
    <div class="spacer" />
    <select bind:value={level} on:change={(e) => changeLevel(e.target.value)} aria-label="难度">
      {#each ['A1','A2','B1','B2','C1','C2'] as l}<option value={l}>{l}</option>{/each}
    </select>
  </div>

  <div class="content reader" use:swipe on:swipeleft={() => nav(1)} on:swiperight={() => nav(-1)}>
    <div class="page-pad reader-pad slide-{slideDir}">
      {#if loadingPara}
        <div class="muted row"><span class="spinner" /> <span>加载段落…</span></div>
      {:else if !paragraph}
        <div class="muted">没有段落</div>
      {:else if mode === 'original'}
        <article class="prose">
          <TextSpans text={paragraph.original_text} on:word={onWord} />
        </article>
      {:else}
        {#if loadingSim}
          <div class="muted row"><span class="spinner" /> <span>正在用 {level} 级别简化…首次可能 5-15 秒</span></div>
        {:else if !simpl}
          <div class="cta">
            <p class="muted">这段还没有 {level} 级别简化版。</p>
            <button class="primary" on:click={() => fetchSimplified(false)}>用 {level} 简化</button>
          </div>
        {:else}
          <SentenceList
            sentences={simpl.sentences}
            openIdx={openSentenceIdx}
            analyzing={analyzingIdx}
            analysisCache={analysisCache}
            on:word={onWord}
            on:speak={(e) => playTTS(e.detail)}
            on:analyze={(e) => openAnalysis(e.detail)}
          />
          <div class="footer-actions">
            <button class="ghost small" on:click={regenerate}>重新生成简化</button>
          </div>
        {/if}
      {/if}
    </div>
  </div>

  <div class="navbar">
    <button class="ghost" on:click={() => nav(-1)} disabled={cur <= 1}>‹ 上一段</button>
    <div class="progress muted small">{cur} / {total}</div>
    <button class="ghost" on:click={() => nav(1)} disabled={cur >= total}>下一段 ›</button>
  </div>
</div>

{#if tocOpen && book}
  <Toc {book} {cur} on:close={() => tocOpen = false} on:jump={(e) => jumpTo(e.detail)} />
{/if}

<style>
  .modebar {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.5rem 0.85rem;
    border-bottom: 1px solid var(--border);
    background: var(--bg);
  }
  .tabs { display: flex; background: var(--bg-elev-2); border-radius: 999px; padding: 3px; }
  .tabs button { background: transparent; border: 0; padding: 0.35rem 0.85rem; border-radius: 999px; color: var(--fg-dim); font-size: 0.85rem; }
  .tabs button.active { background: var(--bg); color: var(--fg); box-shadow: 0 1px 4px rgba(0,0,0,0.2); }
  select { width: auto; padding: 0.35rem 0.5rem; font-size: 0.85rem; }

  .reader-pad { padding-top: 1.5rem; padding-bottom: 4rem; }
  .reader-pad.slide-1  { animation: slideIn-r 220ms ease; }
  .reader-pad.slide--1 { animation: slideIn-l 220ms ease; }
  @keyframes slideIn-r { from { transform: translateX(8%); opacity: 0; } to { transform: none; opacity: 1; } }
  @keyframes slideIn-l { from { transform: translateX(-8%); opacity: 0; } to { transform: none; opacity: 1; } }

  .prose {
    font-family: var(--reading-font);
    font-size: 1.15rem;
    line-height: 1.75;
    letter-spacing: 0.005em;
    color: var(--fg);
  }
  .cta { padding: 2rem 0; text-align: center; }

  .footer-actions { margin-top: 2rem; display: flex; justify-content: center; }

  .navbar {
    display: flex; align-items: center; gap: 0.5rem;
    padding: 0.6rem 0.85rem;
    border-top: 1px solid var(--border);
    background: var(--bg);
    position: sticky; bottom: 0;
  }
  .navbar .progress { flex: 1; text-align: center; }

  .title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
  .title .t { font-weight: 600; font-size: 0.95rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .title .sub { color: var(--fg-mute); font-size: 0.75rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
