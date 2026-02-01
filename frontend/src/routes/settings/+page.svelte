<script lang="ts">
	import { t, locale } from 'svelte-i18n';
	import { Auth } from '$lib/api';
	import { user } from '../../user';

	let currentLocale = $state('');
	let saving = $state(false);

	$effect(() => {
		if ($locale) {
			currentLocale = $locale;
		}
	});

	async function changeLocale(newLocale: string) {
		waiting = true;
		try {
			// Update local state immediately for responsiveness
			locale.set(newLocale);
			localStorage.setItem('locale', newLocale);
			currentLocale = newLocale;

			// Update backend if user is logged in
			if ($user) {
				const response = await Auth.updateLanguage({ language: newLocale });
				if (response.user) {
					user.set(response.user);
				}
			}
		} catch (e) {
			console.error('Failed to update language', e);
			// Optionally revert or show error
		} finally {
			waiting = false;
		}
	}

	let waiting = $state(false);
</script>

<svelte:head>
	<title>{$t('settings.title')}</title>
</svelte:head>

<div class="mx-auto w-full max-w-2xl">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body">
			<h2 class="card-title">{$t('settings.title')}</h2>

			<div class="form-control w-full max-w-xs mt-4">
				<label class="label" for="language-select">
					<span class="label-text">{$t('settings.language')}</span>
				</label>
				<select
					class="select select-bordered"
					id="language-select"
					value={currentLocale}
					onchange={(e) => changeLocale(e.currentTarget.value)}
					disabled={waiting}
				>
					<option value="en">English</option>
					<option value="ru">Русский</option>
				</select>
			</div>
		</div>
	</div>
</div>
