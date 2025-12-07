<script lang="ts">
    import type {GreetResponse} from "$lib/gen/api/v1/greet_pb";
    import {Client} from "$lib/client";
    import {page} from "$app/state";
    import {onMount} from "svelte";

    let name = $state("");
    let result = $state<Promise<GreetResponse>>();

    onMount(() => {
        name =  page.url.searchParams.get("name") ?? "World";
    });

    async function greet() {
        result = Client.greet({"name": name})
    }
</script>

<svelte:head>
    <title>Say hello</title>
</svelte:head>

<style>
    .lang {
        font-size: 0.8em;
    }
</style>
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
            {#if value.language}
                <span class="lang"> - {value.language}</span>
            {/if}
        </blockquote>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}
