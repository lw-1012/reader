<script>
  import { createEventDispatcher } from 'svelte';
  export let text = '';
  const dispatch = createEventDispatcher();

  $: tokens = tokenize(text);

  function tokenize(s) {
    const re = /([A-Za-z][A-Za-z'\-]*)|(\s+)|([^A-Za-z\s]+)/g;
    const out = [];
    let m;
    while ((m = re.exec(s)) !== null) {
      if (m[1]) out.push({ type: 'word', value: m[1] });
      else if (m[2]) out.push({ type: 'space', value: m[2] });
      else out.push({ type: 'punct', value: m[3] });
    }
    return out;
  }

  function tap(word) {
    dispatch('word', word);
  }
</script>

{#each tokens as t, i (i)}
  {#if t.type === 'word'}
    <span class="w" role="button" tabindex="0"
      on:click|stopPropagation={() => tap(t.value)}
      on:keydown={(e) => e.key === 'Enter' && tap(t.value)}>{t.value}</span>
  {:else if t.type === 'space'}{t.value}{:else}<span class="p">{t.value}</span>{/if}
{/each}

<style>
  .w {
    cursor: pointer;
    border-radius: 4px;
    padding: 0 1px;
    transition: background 90ms ease;
  }
  .w:active, .w:focus { background: rgba(125, 211, 252, 0.25); outline: none; }
  @media (hover: hover) {
    .w:hover { background: rgba(125, 211, 252, 0.15); }
  }
  .p { color: inherit; }
</style>
