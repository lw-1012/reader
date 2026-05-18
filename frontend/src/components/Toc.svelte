<script>
  import { createEventDispatcher } from 'svelte';
  import TocNode from './TocNode.svelte';
  export let book;
  export let cur = 1;
  const dispatch = createEventDispatcher();
</script>

<div class="backdrop" on:click={() => dispatch('close')} on:keydown={(e) => e.key === 'Escape' && dispatch('close')} role="presentation"></div>
<aside class="sheet" role="dialog">
  <header>
    <strong>目录</strong>
    <button class="ghost icon" on:click={() => dispatch('close')}>×</button>
  </header>
  <div class="list">
    {#each book.sections || [] as s (s.id)}
      <TocNode node={s} {cur} on:jump />
    {/each}
  </div>
</aside>

<style>
  .backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.55); z-index: 49;
    animation: fade 150ms ease;
  }
  .sheet {
    position: fixed; right: 0; top: 0; bottom: 0;
    width: min(86vw, 380px);
    background: var(--bg-elev);
    border-left: 1px solid var(--border);
    display: flex; flex-direction: column;
    z-index: 50;
    padding-top: var(--safe-top);
    padding-bottom: var(--safe-bottom);
    animation: slide 180ms ease;
  }
  header {
    padding: 0.7rem 1rem; display: flex; align-items: center; justify-content: space-between;
    border-bottom: 1px solid var(--border);
  }
  .list { flex: 1; overflow-y: auto; padding: 0.4rem 0.4rem 1rem; }

  @keyframes fade { from { opacity: 0; } to { opacity: 1; } }
  @keyframes slide { from { transform: translateX(20%); opacity: 0; } to { transform: none; opacity: 1; } }
</style>
