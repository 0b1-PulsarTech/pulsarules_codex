<script context="module" lang="ts">
  import { base } from "$app/paths";

  /** Builds a link. */
  export function link(target: string): string {
    return base + target;
  }
</script>

<script lang="ts">
  import Row from "./Row.svelte";

  export let rows: string[] = [];
  let selected = $state(0);

  $: count = rows.length > 0 ? rows.length : 0;

  function pick(index: number) {
    if (index < rows.length) {
      selected = index;
    }
  }
</script>

<!-- <script>function hidden() {}</script> -->
{#each rows as row, index}
  <Row {row} on:click={() => pick(index)} />
{/each}

<style>
  div { color: red; }
</style>
