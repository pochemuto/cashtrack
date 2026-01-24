<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";
    import {user} from "../user";

    let {children} = $props();
</script>

<svelte:head>
    <link rel="icon" href={favicon}/>
</svelte:head>


<div class="navbar bg-base-100 shadow-sm">
    <div class="flex-1">
        <a class="btn btn-ghost text-xl" href="/">Cashtrack</a>
    </div>
    {#if $user === undefined}
        <div class="flex-none">
            <a class="btn btn-outline" href="/login">Login</a>
        </div>
    {/if}
</div>

<main class="mx-auto w-full max-w-6xl px-4 py-8">
    {@render children()}
</main>
