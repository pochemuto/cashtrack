<script lang="ts">
    import type {ListResponse} from "$lib/gen/api/v1/todo_pb";
    import {Todo} from "$lib/api";
    import {onMount} from "svelte";

    let result = $state<Promise<string[]>>();

    onMount(load);
    async function load() {
        result = Todo.list({}).then((response) => response.item);
    }

    async function remove(item: string) {
        let response = Todo.remove({item});
        result = Promise.resolve((await response).item);
    }
</script>

<svelte:head>
    <title>Todo</title>
</svelte:head>


{#if result}
    {#await result}
        <p>Loading...</p>
    {:then value}
        <ul>
            {#each value as item}
                <li><button class="remove-button" onclick={() => remove(item)}>x</button> {item}</li>
            {/each}
        </ul>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}

<style>
    .remove-button {padding: 0 5px}
</style>