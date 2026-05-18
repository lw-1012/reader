<script>
  import { createEventDispatcher } from 'svelte';
  export let node;
  export let cur = 1;
  const dispatch = createEventDispatcher();

  let expanded = node.depth === 0;
  $: active = node.para_from && cur >= node.para_from && cur <= node.para_to;
  $: hasChildren = node.children && node.children.length > 0;
</script>

<div class="item" style="padding-left:{node.depth * 0.9}rem">
  <button class="row-btn" class:active on:click={() => node.para_from && dispatch('jump', node.para_from)}>
    {#if hasChildren}
      <span class="caret" on:click|stopPropagation={() => expanded = !expanded} role="button" tabindex="-1">{expanded ? '▾' : '▸'}</span>
    {:else}
      <span class="caret-empty"></span>
    {/if}
    <span class="t">{node.title || '(无标题)'}</span>
    {#if node.para_from}<span class="meta">{node.para_from}-{node.para_to}</span>{/if}
  </button>
  {#if expanded && hasChildren}
    {#each node.children as c (c.id)}
      <svelte:self node={c} {cur} on:jump />
    {/each}
  {/if}
</div>

<style>
  .item { margin: 0.05rem 0; }
  .row-btn {
    width: 100%; display: flex; gap: 0.4rem; align-items: center;
    background: transparent; border: 0; padding: 0.45rem 0.5rem;
    border-radius: 8px; text-align: left; color: var(--fg);
  }
  .row-btn:active { background: var(--bg-elev-2); }
  .row-btn.active { background: var(--bg-elev-2); color: var(--accent); }
  .caret, .caret-empty { width: 1em; color: var(--fg-mute); display: inline-block; }
  .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.9rem; }
  .meta { color: var(--fg-mute); font-size: 0.72rem; }
</style>
