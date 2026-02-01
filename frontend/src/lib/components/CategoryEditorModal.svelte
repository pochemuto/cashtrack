<script lang="ts">
	import { onMount } from 'svelte';
	import type { Category } from '$lib/gen/api/v1/categories_pb';
	import CategoryColorPicker from '$lib/components/CategoryColorPicker.svelte';
	import CategoryBadge from '$lib/components/CategoryBadge.svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		open?: boolean;
		title?: string;
		name?: string;
		color?: string | null;
		categories?: Category[];
		parentId?: number | null;
		isGroup?: boolean;
		selfId?: number | null;
		showParent?: boolean;
		showGroupToggle?: boolean;
		confirmLabel?: string;
		saving?: boolean;
		oncancel?: () => void;
		onsave?: () => void;
	}

	let {
		open = false,
		title = '',
		name = $bindable(''),
		color = $bindable(null),
		categories = [],
		parentId = $bindable(null),
		isGroup = $bindable(false),
		selfId = null,
		showParent = true,
		showGroupToggle = true,
		confirmLabel = '',
		saving = false,
		oncancel,
		onsave
	}: Props = $props();

	let nameInput: { focus: () => void } | null = $state(null);
	let parentOptions = $derived(categories.filter((category) => category.id !== selfId));
	let parentIdValue = $derived(parentId ? String(parentId) : '');

	function handleCancel() {
		oncancel?.();
	}

	function handleSave() {
		onsave?.();
	}

	function handleParentChange(event: Event) {
		const value = (event.currentTarget as HTMLSelectElement).value;
		parentId = value ? Number(value) : null;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!open) {
			return;
		}
		if (event.key === 'Escape') {
			event.preventDefault();
			handleCancel();
		}
	}

	$effect(() => {
		if (open) {
			queueMicrotask(() => nameInput?.focus());
		}
	});

	$effect(() => {
		if (parentId !== null && !parentOptions.some((category) => category.id === parentId)) {
			parentId = null;
		}
	});

	$effect(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

{#if open}
	<div
		class="modal modal-open"
		role="dialog"
		aria-modal="true"
		aria-labelledby="category-editor-title"
	>
		<div class="modal-box">
			<h3 id="category-editor-title" class="text-lg font-semibold">
				{title || $t('categories.modalTitleEdit')}
			</h3>
			<form
				class="mt-4 space-y-4"
				onsubmit={(e) => {
					e.preventDefault();
					handleSave();
				}}
			>
				<div class="form-control w-full">
					<CategoryBadge
						bind:this={nameInput}
						bind:name
						color={color ?? ''}
						editable={true}
						placeholder={$t('categories.namePlaceholder')}
						className="badge-lg"
					/>
				</div>
				<div class="form-control w-full">
					<div class="label">
						<button
							class="btn btn-ghost btn-xs"
							type="button"
							onclick={() => (color = null)}
							disabled={!color}
						>
							{$t('categories.noColor')}
						</button>
					</div>
					<CategoryColorPicker
						bind:hex={color}
						label={$t('categories.color')}
						inline={true}
						nullable={false}
					/>
				</div>
				{#if showParent}
					<div class="form-control w-full">
						<label class="label" for="parent-category-select">
							<span class="label-text">{$t('categories.parentLabel')}</span>
						</label>
						<select
							class="select select-bordered"
							id="parent-category-select"
							bind:value={parentId}
							onchange={handleParentChange}
							disabled={!parentOptions.length}
						>
							<option value="">{$t('categories.noParent')}</option>
							{#each parentOptions as category}
								<option value={String(category.id)}>
									{category.name}{category.isGroup ? ` (${$t('categories.group')})` : ''}
								</option>
							{/each}
						</select>
					</div>
				{/if}
				{#if showGroupToggle}
					<label class="label cursor-pointer gap-2">
						<input class="checkbox checkbox-sm" type="checkbox" bind:checked={isGroup} />
						<span class="label-text">{$t('categories.groupToggle')}</span>
					</label>
				{/if}
				<div class="modal-action">
					<button class="btn btn-primary" type="submit" disabled={saving || !name.trim()}>
						{confirmLabel || $t('common.save')}
					</button>
					<button class="btn btn-ghost" type="button" onclick={handleCancel}>
						{$t('common.cancel')}
					</button>
				</div>
			</form>
		</div>
		<button class="modal-backdrop" type="button" onclick={handleCancel} aria-label="close"></button>
	</div>
{/if}
