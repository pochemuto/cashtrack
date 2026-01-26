<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";
    import {loadUser, logoutUser, user} from "../user";
    import {onMount} from "svelte";
    import {cancelGoogleSignIn, initializeGoogleSignIn} from "$lib/auth/google";
    import {resolveApiUrl} from "$lib/url";

    let {children} = $props();

    const GOOGLE_CLIENT_ID = "1010772966942-khflv7f816n0bqebf7mll7hb0eu589r0.apps.googleusercontent.com";

    declare global {
        interface Window {
            google?: typeof google;
        }
    }

    function handleGoogleCredential(response: google.accounts.id.CredentialResponse) {
        console.info("Google credential received", response);
        const redirect = window.location.origin;
        const authUrl = resolveApiUrl(
            `auth?credential=${encodeURIComponent(response.credential)}&redirect=${encodeURIComponent(redirect)}`
        );
        window.location.href = authUrl;
    }

    onMount(() => {
        let destroyed = false;
        let promptRequested = false;

        void loadUser();

        const tryInit = () => {
            if (destroyed || !promptRequested) {
                return;
            }
            if (initializeGoogleSignIn(null, GOOGLE_CLIENT_ID, handleGoogleCredential, {renderButton: false})) {
                return;
            }
            requestAnimationFrame(tryInit);
        };

        const unsubscribe = user.subscribe((value) => {
            if (value === undefined) {
                if (!promptRequested) {
                    promptRequested = true;
                    tryInit();
                }
                return;
            }
            promptRequested = false;
            cancelGoogleSignIn();
        });

        return () => {
            destroyed = true;
            cancelGoogleSignIn();
            unsubscribe();
        };
    });
</script>

<svelte:head>
    <link rel="icon" href={favicon}/>
    <script src="https://accounts.google.com/gsi/client" async defer></script>
</svelte:head>


<div class="navbar bg-base-100 shadow-sm">
    <div class="flex-1 flex items-center gap-2">
        <a class="link link-hover text-xl pl-2" href="/">Cashtrack</a>
        {#if $user}
            <a class="link link-hover px-2" href="/transactions">Transactions</a>
            <a class="link link-hover px-2" href="/categories">Categories</a>
            <a class="link link-hover px-2" href="/import">Import</a>
        {/if}
    </div>
    {#if $user}
        <div class="dropdown dropdown-end">
            <div tabindex="0" role="button" class="btn btn-ghost">
                {$user.username}
            </div>
            <ul class="menu dropdown-content mt-2 w-40 rounded-box bg-base-100 p-2 shadow">
                <li>
                    <button type="button" onclick={() => logoutUser()}>Logout</button>
                </li>
            </ul>
        </div>
    {:else if $user === undefined}
        <div class="flex-none">
            <a class="btn btn-outline" href="/login">Login</a>
        </div>
    {/if}
</div>

<main class="w-full px-4 py-8">
    {@render children()}
</main>
