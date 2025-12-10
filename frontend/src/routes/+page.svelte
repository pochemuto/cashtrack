<script lang="ts">
    import {Todo} from "$lib/api";
    import type {ListItem} from "$lib/gen/api/v1/todo_pb"
    import {onMount} from "svelte";

    let result = $state<Promise<ListItem[]>>();
    let newItem = $state("");
    let input: HTMLInputElement;

    onMount(load);

    async function load() {
        result = Todo.list({}).then((response) => response.items);
    }

    async function remove(id: number) {
        let response = Todo.remove({id});
        result = Promise.resolve((await response).items);
    }

    async function add() {
        if (!newItem) {
            return;
        }
        let response = await Todo.add({items: [{title: newItem}]});
        result = Promise.resolve(response.items);
        newItem = "";
        input.focus();
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
                <li>
                    <button class="remove-button" onclick={() => remove(item.id)}>x</button> {item.title}</li>
            {/each}
        </ul>
        <form onsubmit="{add}">
        <input type="text" bind:value={newItem} bind:this={input}>
        <button type="submit">+</button>
        </form>
    {:catch error}
        <!-- promise was rejected -->
        <p>Something went wrong: {error.message}</p>
    {/await}
{/if}

<style>
    .remove-button {
        padding: 0 5px
    }
</style>