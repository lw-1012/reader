<script>
  import { createEventDispatcher } from 'svelte';
  import TextSpans from './TextSpans.svelte';

  export let sentences = [];
  export let openIdx = null;
  export let analyzing = null;
  export let analysisCache = {};

  const dispatch = createEventDispatcher();
</script>

<article class="sents">
  {#each sentences as s, i (i)}
    <div class="sent" class:open={openIdx === i}>
      <div class="line">
        <span class="text">
          <TextSpans text={s} on:word={(e) => dispatch('word', e.detail)} />
        </span>
        <span class="icons">
          <button class="ghost icon" on:click={() => dispatch('speak', s)} title="朗读句子">🔊</button>
          <button class="ghost icon" on:click={() => dispatch('analyze', i)} title="分析" class:active={openIdx === i}>💡</button>
        </span>
      </div>

      {#if openIdx === i}
        <div class="analysis">
          {#if analyzing === i && !analysisCache[s]}
            <div class="row muted small"><span class="spinner" /> <span>分析中…</span></div>
          {:else if analysisCache[s]}
            {@const a = analysisCache[s]}
            <div class="tr">{a.translation}</div>
            {#if a.vocab && a.vocab.length}
              <div class="vocab">
                {#each a.vocab as v}
                  <div class="vrow">
                    <button class="vword" on:click={() => dispatch('word', v.word)} title="朗读">{v.word}</button>
                    <span class="vpos">{v.pos}</span>
                    <span class="vmean">{v.meaning}</span>
                  </div>
                {/each}
              </div>
            {/if}
            {#if a.grammar}
              <div class="grammar"><span class="lbl">语法</span> {a.grammar}</div>
            {/if}
            {#if a.notes}
              <div class="notes"><span class="lbl">备注</span> {a.notes}</div>
            {/if}
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</article>

<style>
  .sents { font-family: var(--reading-font); font-size: 1.12rem; line-height: 1.7; }
  .sent { margin-bottom: 0.4rem; padding: 0.3rem 0.1rem; border-radius: 8px; transition: background 120ms ease; }
  .sent.open { background: var(--bg-elev); }
  .line { display: inline; }
  .text { }
  .icons { white-space: nowrap; display: inline-flex; vertical-align: middle; gap: 0.15rem; margin-left: 0.25rem; }
  .icons :global(button.icon) { padding: 0.1rem 0.35rem; min-width: 0; min-height: 0; font-size: 0.95rem; opacity: 0.55; }
  .icons :global(button.icon:hover) { opacity: 1; }
  .icons :global(button.icon.active) { opacity: 1; background: var(--bg-elev-2); }

  .analysis {
    margin-top: 0.5rem;
    padding: 0.7rem 0.85rem;
    background: var(--bg-elev-2);
    border-left: 3px solid var(--accent);
    border-radius: 0 8px 8px 0;
    font-family: var(--ui-font);
    font-size: 0.92rem;
    line-height: 1.55;
  }
  .tr { color: var(--fg); font-weight: 500; }
  .vocab { margin-top: 0.6rem; display: grid; gap: 0.25rem; }
  .vrow { display: flex; gap: 0.5rem; align-items: baseline; }
  .vword { background: var(--bg); border: 1px solid var(--border); border-radius: 6px; padding: 0.05rem 0.45rem; font-family: var(--reading-font); font-weight: 500; cursor: pointer; }
  .vpos { color: var(--fg-mute); font-size: 0.78rem; }
  .vmean { color: var(--fg-dim); flex: 1; min-width: 0; }
  .grammar, .notes { margin-top: 0.5rem; color: var(--fg-dim); }
  .lbl { display: inline-block; min-width: 2.5em; color: var(--fg-mute); font-size: 0.78rem; margin-right: 0.4rem; }
</style>
