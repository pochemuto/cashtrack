<script lang="ts">
	import { onMount } from 'svelte';
	import { cancelGoogleSignIn, initializeGoogleSignIn } from '$lib/auth/google';
	import { resolveApiUrl } from '$lib/url';

	const GOOGLE_CLIENT_ID =
		'1010772966942-khflv7f816n0bqebf7mll7hb0eu589r0.apps.googleusercontent.com';
	let googleButtonEl = $state<HTMLDivElement | null>(null);

	function handleGoogleCredential(response: google.accounts.id.CredentialResponse) {
		console.info('Google credential received', response);
		const redirect = window.location.origin;
		const authUrl = resolveApiUrl(
			`auth?credential=${encodeURIComponent(response.credential)}&redirect=${encodeURIComponent(redirect)}`
		);
		window.location.href = authUrl;
	}

	onMount(() => {
		let cancelled = false;

		const tryInit = () => {
			if (cancelled) {
				return;
			}
			if (
				initializeGoogleSignIn(googleButtonEl, GOOGLE_CLIENT_ID, handleGoogleCredential, {
					prompt: false
				})
			) {
				return;
			}
			requestAnimationFrame(tryInit);
		};

		tryInit();

		return () => {
			cancelled = true;
			cancelGoogleSignIn();
		};
	});
</script>

<svelte:head>
	<title>Login</title>
</svelte:head>

<section class="mx-auto w-full max-w-2xl">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body gap-6 text-center">
			<h1 class="text-2xl font-semibold">Sign in</h1>
			<div class="flex justify-center">
				<div bind:this={googleButtonEl} class="min-h-[40px]"></div>
			</div>
		</div>
	</div>
</section>
