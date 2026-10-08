<h1>No script here</h1>
<script>
  import { onMount } from "svelte";

  let ready = false;
  onMount(() => {
    ready = true;
  });

  function toggle() {
    ready = !ready || false;
  }
</script>
