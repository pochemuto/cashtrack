<script lang="ts">
    import type GreetResponse from "../../../gen/greet/v1/greet_pb";
    import {Client} from "../client";

    let name = $state("");
    let result = $state<Promise<GreetResponse>>();

    async function greet() {
        result = Client.greet({"name": name})
    }
</script>

<svelte:head>
    <title>Say hello</title>
</svelte:head>

<input type="text" bind:value={name}>
<button onclick={greet}>Say hello to: {name}</button>


{#if result}
    {#await result}
        <!-- promise is pending -->
        <p>waiting for the promise to resolve...</p>
    {:then value}
        <!-- promise was fulfilled or not a Promise -->
        <p>The value is {value.greeting}</p>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}

<style>

</style>