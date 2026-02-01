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
				listError = 'Не удалось загрузить категории.';
			}

			rules = rulesResponse.rules ?? [];
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				listError = 'Нужен вход для просмотра правил.';
				rules = [];
				return;
			}
			listError = 'Не удалось загрузить категории.';
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
				actionError = 'Не удалось добавить категорию.';
				return;
			}
			addCategory(response.category);
			cancelCategoryEdit();
		} catch {
			actionError = 'Не удалось добавить категорию.';
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
			actionError = 'Не удалось обновить категорию.';
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
			actionError = 'Не удалось удалить категорию.';
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
				actionError = 'Не удалось добавить правило.';
				return;
			}
			rules = [...rules, response.rule];
			newRuleText = '';
			newRuleCategoryId = '';
		} catch {
			actionError = 'Не удалось добавить правило.';
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
			actionError = 'Не удалось обновить правило.';
		}
	}

	async function deleteRule(ruleId: number) {
		actionError = '';
		menuOpen = null;
		try {
			await Categories.deleteCategoryRule({ id: ruleId });
			rules = rules.filter((rule) => rule.id !== ruleId);
		} catch {
			actionError = 'Не удалось удалить правило.';
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
			actionError = 'Не удалось изменить порядок правил.';
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
			showToast(`Правила применены. Обновлено транзакций: ${updatedCount}.`);
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				actionError = 'Нужен вход для применения правил.';
				return;
			}
			actionError = 'Не удалось применить правила.';
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
	<title>Categories</title>
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
				<h1 class="text-2xl font-semibold">Категории</h1>
				<button class="btn btn-primary" type="button" onclick={openCreateCategory}>
					Добавить
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
				<div class="text-sm opacity-70">Загрузка категорий...</div>
			{:else if $categories.length === 0}
				<div class="text-sm opacity-70">Категории пока не добавлены.</div>
			{:else}
				<div class="space-y-2">
					{#each $categories as category}
						<div class="flex items-center gap-2">
							<button
								class="btn btn-ghost btn-xs"
								type="button"
								onclick={() => requestDeleteCategory(category)}
								aria-label="Удалить категорию"
								title="Удалить категорию"
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
											<span>Группа</span>
										{/if}
										{#if category.parentId}
											<span>
												{category.isGroup ? ' · ' : ''}
												Родитель: {categoryMap.get(category.parentId) || `#${category.parentId}`}
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
				<h2 class="text-xl font-semibold">Правила категоризации</h2>
				<p class="text-sm opacity-70">Пока поддерживается правило "описание содержит".</p>
			</div>

			<div class="flex flex-wrap items-center gap-3">
				<label class="label cursor-pointer gap-2 p-0">
					<input class="checkbox checkbox-sm" type="checkbox" bind:checked={applyRulesToAll} />
					<span class="label-text">Применить ко всем транзакциям</span>
				</label>
				<button
					class="btn btn-outline btn-sm"
					type="button"
					onclick={applyRules}
					disabled={applyingRules || rules.length === 0}
				>
					{applyingRules ? 'Применение...' : 'Применить правила'}
				</button>
			</div>

			<div class="grid gap-3 lg:grid-cols-[minmax(200px,1fr)_minmax(240px,2fr)_auto]">
				<select class="select select-bordered" bind:value={newRuleCategoryId}>
					<option value="" disabled>Категория</option>
					{#each ruleCategories as category}
						<option value={category.id}>{category.name}</option>
					{/each}
				</select>
				<input
					class="input input-bordered"
					type="text"
					placeholder="Описание содержит"
					bind:value={newRuleText}
				/>
				<button
					class="btn btn-primary"
					type="button"
					onclick={createRule}
					disabled={!ruleCategories.length}
				>
					Добавить
				</button>
			</div>

			{#if loading}
				<div class="text-sm opacity-70">Загрузка правил...</div>
			{:else if rules.length === 0}
				<div class="text-sm opacity-70">Правила пока не настроены.</div>
			{:else}
				<div class="overflow-x-auto overflow-y-visible">
					<table class="table">
						<thead>
							<tr>
								<th>Категория</th>
								<th>Описание содержит</th>
								<th class="text-right">Действия</th>
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
													Сохранить
												</button>
												<button class="btn btn-sm btn-ghost" type="button" onclick={cancelRuleEdit}>
													Отмена
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
	title={editorMode === 'create' ? 'Новая категория' : 'Редактировать категорию'}
	confirmLabel={editorMode === 'create' ? 'Добавить' : 'Сохранить'}
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
			<h3 id="delete-category-title" class="text-lg font-semibold">Удалить категорию?</h3>
			<p class="mt-3 text-sm opacity-80">
				Категория “{deleteCategoryName}” будет удалена без возможности восстановления.
			</p>
			<div class="modal-action">
				<button class="btn btn-error" type="button" onclick={confirmDeleteCategory}>
					Удалить
				</button>
				<button class="btn btn-ghost" type="button" onclick={cancelDeleteCategory}> Отмена </button>
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
					Редактировать
				</button>
			</li>
			<li>
				<button type="button" onclick={() => menuOpen && deleteCategory(menuOpen.id)}>
					Удалить
				</button>
			</li>
		{:else}
			<li>
				<button type="button" onclick={() => menuOpen && startRuleEditById(menuOpen.id)}>
					Редактировать
				</button>
			</li>
			<li>
				<button type="button" onclick={() => menuOpen && deleteRule(menuOpen.id)}> Удалить </button>
			</li>
		{/if}
	</ul>
{/if}
```
