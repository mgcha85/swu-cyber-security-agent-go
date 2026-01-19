<script lang="ts">
    import { marked } from "marked";
    import { onMount } from "svelte";

    export let markdown = "";

    let html = "";

    $: {
        try {
            html = marked.parse(markdown) as string;
        } catch (e) {
            html = '<p class="text-red-500">Error rendering markdown</p>';
        }
    }
</script>

<div
    class="prose prose-invert max-w-none prose-headings:text-cyan-400 prose-a:text-blue-400 prose-strong:text-white prose-code:text-yellow-300"
>
    {@html html}
</div>
