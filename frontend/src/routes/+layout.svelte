<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";
    import {user} from "../user";
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
        window.location.href = resolveApiUrl("auth");
    }

    onMount(() => {
        let destroyed = false;
        let promptRequested = false;

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
