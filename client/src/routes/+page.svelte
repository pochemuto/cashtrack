<script lang="ts">
    import { Client } from "../client";

    let message = $state("Nothing");
    let name = $state("");
    async function greet() {
        let res = await Client.greet({"name": name});
        message = res.greeting;
    }
</script>

<svelte:head>
    <title>Say hello</title>
</svelte:head>

<input type="text" bind:value={name}>
<button onclick={greet}>Say hello to: {name}</button>

<p>{message}</p>

<style>
</style>