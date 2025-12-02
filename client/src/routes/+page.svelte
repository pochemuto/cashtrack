<script lang="ts">
    import type {GreetResponse} from "../gen/greet/v1/greet_pb";
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

<form>
    <label for="name">Name: </label>
    <input type="text" name="name" bind:value={name}>
    <button onclick={greet}>Say hello for: {name}</button>
</form>


{#if result}
    {#await result}
        <!-- promise is pending -->
        <p>waiting for the promise to resolve...</p>
    {:then value}
        <blockquote>
            <!-- promise was fulfilled or not a Promise -->
            <p>{value.greeting}</p>
        </blockquote>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}
