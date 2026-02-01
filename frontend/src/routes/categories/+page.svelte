<script lang="ts">
	import { onMount } from 'svelte';
	import { Categories } from '$lib/api';
	import type { Category, CategoryRule } from '$lib/gen/api/v1/categories_pb';
	import { Code, ConnectError } from '@connectrpc/connect';
	import {
		categories,
		addCategory,
		loadCategories,
		removeCategory,
		updateCategory
	} from '$lib/stores/categories';
	import { user } from '../../user';
	import CategoryBadge from '$lib/components/CategoryBadge.svelte';
	import CategoryEditorModal from '$lib/components/CategoryEditorModal.svelte';
	import { t } from 'svelte-i18n';

	let rules = $state<CategoryRule[]>([]);
	let loading = $state(false);
	let listError = $state('');
	let actionError = $state('');
	let toastMessage = $state('');
	let toastTimeout: ReturnType<typeof setTimeout> | null = null;

	let editorOpen = $state(false);
	let editorMode = $state<'create' | 'edit'>('create');
	let editorCategoryId = $state<number | null>(null);
	let editorName = $state('');
	let editorColor = $state<string | null>(null);
	let editorParentId = $state<number | null>(null);
	let editorIsGroup = $state(false);

	let newRuleCategoryId = $state('');
	let newRuleText = $state('');
	let editingRuleId = $state<number | null>(null);
	let editingRuleCategoryId = $state('');
	let editingRuleText = $state('');
	let applyRulesToAll = $state(false);
	let applyingRules = $state(false);
	let rulesReordering = $state(false);
	let draggingRuleIndex = $state<number | null>(null);
	let dragOverIndex = $state<number | null>(null);
	let menuOpen = $state<
		| { type: 'category'; id: number; x: number; y: number }
		| { type: 'rule'; id: number; x: number; y: number }
		| null
	>(null);
	let menuElement: HTMLUListElement | null = $state(null);
	let menuAnchor: HTMLElement | null = $state(null);
	let lastUserId: number | null = $state(null);
	let deleteModalOpen = $state(false);
	let deleteCategoryId = $state<number | null>(null);
	let deleteCategoryName = $state('');

	let categoryMap = $derived(new Map($categories.map((category) => [category.id, category.name])));
	let ruleCategories = $derived($categories.filter((category) => !category.isGroup));

	async function loadData() {
		if (!$user || !$user.id) {
			await loadCategories();
			rules = [];
			listError = '';
			loading = false;
			return;
		}

		loading = true;
		listError = '';

		try {
			const [categoriesOk, rulesResponse] = await Promise.all([
				loadCategories(),
				Categories.listCategoryRules({})
			]);

			if (!categoriesOk) {
				listError = $t('categories.errorList');
			}

			rules = rulesResponse.rules ?? [];
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				listError = $t('rules.loginRequired');
				rules = [];
				return;
			}
			listError = $t('categories.errorList');
			rules = [];
		} finally {
			loading = false;
		}
	}

	async function createCategory() {
		actionError = '';
		const name = editorName.trim();
		if (!name) {
			return;
		}
		const color = (editorColor ?? '').trim();

		try {
			const response = await Categories.createCategory({
				name,
				color,
				parentId: editorParentId ?? 0,
				isGroup: editorIsGroup
			});
			if (!response.category) {
				actionError = $t('categories.errorAction');
				return;
			}
			addCategory(response.category);
			cancelCategoryEdit();
		} catch {
			actionError = $t('categories.errorAction');
		}
	}

	function openCategoryEditor(category: Category) {
		editorMode = 'edit';
		editorCategoryId = category.id;
		editorName = category.name;
		editorColor = category.color || null;
		editorParentId = category.parentId ? category.parentId : null;
		editorIsGroup = category.isGroup;
		editorOpen = true;
		menuOpen = null;
	}

	function openCategoryEditorById(categoryId: number) {
		const category = $categories.find((item) => item.id === categoryId);
		if (!category) {
			return;
		}
		openCategoryEditor(category);
	}

	function cancelCategoryEdit() {
		editorOpen = false;
		editorMode = 'create';
		editorCategoryId = null;
		editorName = '';
		editorColor = null;
		editorParentId = null;
		editorIsGroup = false;
	}

	async function saveCategory(categoryId: number) {
		actionError = '';
		const name = editorName.trim();
		if (!name) {
			return;
		}
		const color = (editorColor ?? '').trim();

		try {
			await Categories.updateCategory({
				id: categoryId,
				name,
				color,
				parentId: editorParentId ?? 0,
				isGroup: editorIsGroup
			});
			const existing = $categories.find((category) => category.id === categoryId);
			if (existing) {
				updateCategory({
					...existing,
					name,
					color,
					parentId: editorParentId ?? 0,
					isGroup: editorIsGroup
				});
			}
			cancelCategoryEdit();
		} catch {
			actionError = $t('categories.errorAction');
		}
	}

	async function deleteCategory(categoryId: number) {
		actionError = '';
		menuOpen = null;
		try {
			await Categories.deleteCategory({ id: categoryId });
			removeCategory(categoryId);
			rules = rules.filter((rule) => rule.categoryId !== categoryId);
		} catch {
			actionError = $t('categories.errorAction');
		}
	}

	function requestDeleteCategory(category: Category) {
		deleteCategoryId = category.id;
		deleteCategoryName = category.name;
		deleteModalOpen = true;
	}

	function cancelDeleteCategory() {
		deleteModalOpen = false;
		deleteCategoryId = null;
		deleteCategoryName = '';
	}

	async function confirmDeleteCategory() {
		if (deleteCategoryId === null) {
			return;
		}
		const categoryId = deleteCategoryId;
		cancelDeleteCategory();
		await deleteCategory(categoryId);
	}

	function openCreateCategory() {
		editorMode = 'create';
		editorCategoryId = null;
		editorName = '';
		editorColor = null;
		editorParentId = null;
		editorIsGroup = false;
		editorOpen = true;
	}

	async function handleCategorySave() {
		if (editorMode === 'create') {
			await createCategory();
			return;
		}
		if (editorCategoryId !== null) {
			await saveCategory(editorCategoryId);
		}
	}

	async function createRule() {
		actionError = '';
		const descriptionContains = newRuleText.trim();
		if (!descriptionContains || !newRuleCategoryId) {
			return;
		}

		try {
			const response = await Categories.createCategoryRule({
				categoryId: Number(newRuleCategoryId),
				descriptionContains
			});
			if (!response.rule) {
				actionError = $t('rules.errorAction');
				return;
			}
			rules = [...rules, response.rule];
			newRuleText = '';
			newRuleCategoryId = '';
		} catch {
			actionError = $t('rules.errorAction');
		}
	}

	function startRuleEdit(rule: CategoryRule) {
		editingRuleId = rule.id;
		editingRuleCategoryId = String(rule.categoryId);
		editingRuleText = rule.descriptionContains;
		menuOpen = null;
	}

	function startRuleEditById(ruleId: number) {
		const rule = rules.find((item) => item.id === ruleId);
		if (!rule) {
			return;
		}
		startRuleEdit(rule);
	}

	function cancelRuleEdit() {
		editingRuleId = null;
		editingRuleCategoryId = '';
		editingRuleText = '';
	}

	async function saveRule(ruleId: number) {
		actionError = '';
		const descriptionContains = editingRuleText.trim();
		if (!editingRuleCategoryId || !descriptionContains) {
			return;
		}

		try {
			await Categories.updateCategoryRule({
				id: ruleId,
				categoryId: Number(editingRuleCategoryId),
				descriptionContains
			});
			rules = rules.map((rule) =>
				rule.id === ruleId
					? {
							...rule,
							categoryId: Number(editingRuleCategoryId),
							descriptionContains
						}
					: rule
			);
			cancelRuleEdit();
		} catch {
			actionError = $t('rules.errorAction');
		}
	}

	async function deleteRule(ruleId: number) {
		actionError = '';
		menuOpen = null;
		try {
			await Categories.deleteCategoryRule({ id: ruleId });
			rules = rules.filter((rule) => rule.id !== ruleId);
		} catch {
			actionError = $t('rules.errorAction');
		}
	}

	function handleRuleDragStart(event: DragEvent, index: number) {
		if (rulesReordering || editingRuleId !== null) {
			event.preventDefault();
			return;
		}
		draggingRuleIndex = index;
		dragOverIndex = index;
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = 'move';
			event.dataTransfer.setData('text/plain', String(rules[index]?.id ?? ''));
		}
	}

	function handleRuleDragOver(event: DragEvent, index: number) {
		if (draggingRuleIndex === null) {
			return;
		}
		event.preventDefault();
		dragOverIndex = index;
		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'move';
		}
	}

	function handleRuleDrop(event: DragEvent, index: number) {
		if (draggingRuleIndex === null) {
			return;
		}
		event.preventDefault();
		const fromIndex = draggingRuleIndex;
		draggingRuleIndex = null;
		dragOverIndex = null;
		if (fromIndex === index) {
			return;
		}
		const nextRules = [...rules];
		const [moved] = nextRules.splice(fromIndex, 1);
		nextRules.splice(index, 0, moved);
		rules = nextRules;
		void persistRuleOrder(nextRules);
	}

	function handleRuleDragEnd() {
		draggingRuleIndex = null;
		dragOverIndex = null;
	}

	async function persistRuleOrder(nextRules: CategoryRule[]) {
		actionError = '';
		rulesReordering = true;
		try {
			await Categories.reorderCategoryRules({ ruleIds: nextRules.map((rule) => rule.id) });
		} catch {
			actionError = $t('rules.errorAction');
			await loadData();
		} finally {
			rulesReordering = false;
		}
	}

	async function applyRules() {
		if (applyingRules) {
			return;
		}
		actionError = '';
		applyingRules = true;
		try {
			const response = await Categories.applyCategoryRules({ applyToAll: applyRulesToAll });
			const updatedCount = response.updatedCount ?? 0;
			showToast($t('rules.applied', { values: { count: updatedCount } }));
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				actionError = $t('rules.loginRequired');
				return;
			}
			actionError = $t('rules.errorAction');
		} finally {
			applyingRules = false;
		}
	}

	function showToast(message: string) {
		toastMessage = message;
		if (toastTimeout) {
			clearTimeout(toastTimeout);
		}
		toastTimeout = setTimeout(() => {
			toastMessage = '';
			toastTimeout = null;
		}, 3000);
	}

	function openMenu(event: MouseEvent, type: 'category' | 'rule', id: number) {
		const target = event.currentTarget as HTMLElement;
		const rect = target.getBoundingClientRect();
		menuOpen = { type, id, x: rect.right, y: rect.bottom };
		menuAnchor = target;
	}

	onMount(() => {
		if ($user?.id) {
			lastUserId = $user.id;
			void loadData();
		}

		const handleGlobalClick = (event: MouseEvent) => {
			if (!menuOpen) {
				return;
			}
			const path = event.composedPath();
			if (menuElement && path.includes(menuElement)) {
				return;
			}
			if (menuAnchor && path.includes(menuAnchor)) {
				return;
			}
			menuOpen = null;
		};

		const handleKeyDown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				menuOpen = null;
			}
		};

		window.addEventListener('click', handleGlobalClick, true);
		window.addEventListener('keydown', handleKeyDown);

		return () => {
			window.removeEventListener('click', handleGlobalClick, true);
			window.removeEventListener('keydown', handleKeyDown);
		};
	});

	$effect(() => {
		if ($user?.id && $user.id !== lastUserId) {
			lastUserId = $user.id;
			void loadData();
		}
	});
</script>

<svelte:head>
	<title>{$t('categories.title')}</title>
</svelte:head>

<section class="mx-auto w-full max-w-6xl space-y-6">
	{#if toastMessage}
		<div class="toast toast-top toast-end z-50">
			<div class="alert alert-success">
				<span>{toastMessage}</span>
			</div>
		</div>
	{/if}
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body gap-6">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h1 class="text-2xl font-semibold">{$t('categories.title')}</h1>
				<button class="btn btn-primary" type="button" onclick={openCreateCategory}>
					{$t('common.add')}
				</button>
			</div>

			{#if listError}
				<div class="alert alert-error">
					<span>{listError}</span>
				</div>
			{/if}
			{#if actionError}
				<div class="alert alert-error">
					<span>{actionError}</span>
				</div>
			{/if}

			{#if loading}
				<div class="text-sm opacity-70">{$t('categories.loading')}</div>
			{:else if $categories.length === 0}
				<div class="text-sm opacity-70">{$t('categories.empty')}</div>
			{:else}
				<div class="space-y-2">
					{#each $categories as category}
						<div class="flex items-center gap-2">
							<button
								class="btn btn-ghost btn-xs"
								type="button"
								onclick={() => requestDeleteCategory(category)}
								aria-label={$t('common.delete')}
								title={$t('common.delete')}
							>
								✕
							</button>
							<div class="flex items-center gap-2">
								<button
									class="p-0 cursor-pointer"
									type="button"
									onclick={() => openCategoryEditor(category)}
								>
									<CategoryBadge name={category.name} color={category.color || ''} />
								</button>
								{#if category.isGroup || category.parentId}
									<div class="text-xs opacity-60 whitespace-nowrap">
										{#if category.isGroup}
											<span>{$t('categories.group')}</span>
										{/if}
										{#if category.parentId}
											<span>
												{category.isGroup ? ' · ' : ''}
												{$t('categories.parent')}: {categoryMap.get(category.parentId) ||
													`#${category.parentId}`}
											</span>
										{/if}
									</div>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<div class="card bg-base-100 shadow-xl">
		<div class="card-body gap-6">
			<div class="space-y-2">
				<h2 class="text-xl font-semibold">{$t('rules.title')}</h2>
				<p class="text-sm opacity-70">{$t('rules.description')}</p>
			</div>

			<div class="flex flex-wrap items-center gap-3">
				<label class="label cursor-pointer gap-2 p-0">
					<input class="checkbox checkbox-sm" type="checkbox" bind:checked={applyRulesToAll} />
					<span class="label-text">{$t('rules.applyToAll')}</span>
				</label>
				<button
					class="btn btn-outline btn-sm"
					type="button"
					onclick={applyRules}
					disabled={applyingRules || rules.length === 0}
				>
					{applyingRules ? $t('rules.applying') : $t('rules.applyButton')}
				</button>
			</div>

			<div class="grid gap-3 lg:grid-cols-[minmax(200px,1fr)_minmax(240px,2fr)_auto]">
				<select class="select select-bordered" bind:value={newRuleCategoryId}>
					<option value="" disabled>{$t('rules.categoryPlaceholder')}</option>
					{#each ruleCategories as category}
						<option value={category.id}>{category.name}</option>
					{/each}
				</select>
				<input
					class="input input-bordered"
					type="text"
					placeholder={$t('rules.textPlaceholder')}
					bind:value={newRuleText}
				/>
				<button
					class="btn btn-primary"
					type="button"
					onclick={createRule}
					disabled={!ruleCategories.length}
				>
					{$t('common.add')}
				</button>
			</div>

			{#if loading}
				<div class="text-sm opacity-70">{$t('rules.loading')}</div>
			{:else if rules.length === 0}
				<div class="text-sm opacity-70">{$t('rules.empty')}</div>
			{:else}
				<div class="overflow-x-auto overflow-y-visible">
					<table class="table">
						<thead>
							<tr>
								<th>{$t('rules.categoryPlaceholder')}</th>
								<th>{$t('rules.textPlaceholder')}</th>
								<th class="text-right">{$t('common.actions')}</th>
							</tr>
						</thead>
						<tbody>
							{#each rules as rule, index}
								<tr
									class:bg-base-200={dragOverIndex === index && draggingRuleIndex !== null}
									ondragover={(event) => handleRuleDragOver(event, index)}
									ondrop={(event) => handleRuleDrop(event, index)}
								>
									<td>
										{#if editingRuleId === rule.id}
											<select
												class="select select-bordered select-sm"
												bind:value={editingRuleCategoryId}
											>
												{#each ruleCategories as category}
													<option value={category.id}>{category.name}</option>
												{/each}
											</select>
										{:else}
											<div class="flex items-center gap-2">
												<button
													class="btn btn-ghost btn-xs cursor-grab"
													type="button"
													draggable={!(rulesReordering || editingRuleId !== null)}
													ondragstart={(event) => handleRuleDragStart(event, index)}
													ondragend={handleRuleDragEnd}
													title="Перетащить правило"
													aria-label="Перетащить правило"
												>
													⋮⋮
												</button>
												<div class="font-medium">{categoryMap.get(rule.categoryId) || '—'}</div>
											</div>
										{/if}
									</td>
									<td>
										{#if editingRuleId === rule.id}
											<input
												class="input input-bordered input-sm w-full"
												type="text"
												bind:value={editingRuleText}
											/>
										{:else}
											<span>{rule.descriptionContains}</span>
										{/if}
									</td>
									<td class="text-right">
										{#if editingRuleId === rule.id}
											<div class="flex justify-end gap-2">
												<button
													class="btn btn-sm btn-primary"
													type="button"
													onclick={() => saveRule(rule.id)}
												>
													{$t('common.save')}
												</button>
												<button class="btn btn-sm btn-ghost" type="button" onclick={cancelRuleEdit}>
													{$t('common.cancel')}
												</button>
											</div>
										{:else}
											<button
												class="btn btn-ghost btn-sm"
												type="button"
												onclick={(event) => openMenu(event, 'rule', rule.id)}
											>
												⋮
											</button>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</div>
	</div>
</section>

<CategoryEditorModal
	open={editorOpen}
	bind:name={editorName}
	bind:color={editorColor}
	bind:parentId={editorParentId}
	bind:isGroup={editorIsGroup}
	categories={$categories}
	selfId={editorCategoryId}
	title={editorMode === 'create'
		? $t('categories.modalTitleCreate')
		: $t('categories.modalTitleEdit')}
	confirmLabel={editorMode === 'create' ? $t('common.add') : $t('common.save')}
	onsave={handleCategorySave}
	oncancel={cancelCategoryEdit}
/>

{#if deleteModalOpen}
	<div
		class="modal modal-open"
		role="dialog"
		aria-modal="true"
		aria-labelledby="delete-category-title"
	>
		<div class="modal-box">
			<h3 id="delete-category-title" class="text-lg font-semibold">
				{$t('categories.deleteTitle')}
			</h3>
			<p class="mt-3 text-sm opacity-80">
				{$t('categories.deleteConfirmation', { values: { name: deleteCategoryName } })}
			</p>
			<div class="modal-action">
				<button class="btn btn-error" type="button" onclick={confirmDeleteCategory}>
					{$t('common.delete')}
				</button>
				<button class="btn btn-ghost" type="button" onclick={cancelDeleteCategory}>
					{$t('common.cancel')}
				</button>
			</div>
		</div>
		<button class="modal-backdrop" type="button" onclick={cancelDeleteCategory} aria-label="close"
		></button>
	</div>
{/if}

{#if menuOpen}
	<ul
		bind:this={menuElement}
		class="menu rounded-box bg-base-100 p-2 shadow z-50 w-36"
		style={`position: fixed; top: ${menuOpen.y}px; left: ${menuOpen.x}px; transform: translate(-100%, 0);`}
	>
		{#if menuOpen.type === 'category'}
			<li>
				<button type="button" onclick={() => menuOpen && openCategoryEditorById(menuOpen.id)}>
					{$t('common.edit')}
				</button>
			</li>
			<li>
				<button type="button" onclick={() => menuOpen && deleteCategory(menuOpen.id)}>
					{$t('common.delete')}
				</button>
			</li>
		{:else}
			<li>
				<button type="button" onclick={() => menuOpen && startRuleEditById(menuOpen.id)}>
					{$t('common.edit')}
				</button>
			</li>
			<li>
				<button type="button" onclick={() => menuOpen && deleteRule(menuOpen.id)}>
					{$t('common.delete')}
				</button>
			</li>
		{/if}
	</ul>
{/if}
```
