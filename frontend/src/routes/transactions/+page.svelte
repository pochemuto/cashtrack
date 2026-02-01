<script lang="ts">
	import { onMount } from 'svelte';
	import { Categories, Transactions } from '$lib/api';
	import type { Transaction, TransactionSummary } from '$lib/gen/api/v1/transactions_pb';
	import type { Category } from '$lib/gen/api/v1/categories_pb';
	import { Code, ConnectError } from '@connectrpc/connect';
	import CategoryBadge from '$lib/components/CategoryBadge.svelte';
	import CategoryEditorModal from '$lib/components/CategoryEditorModal.svelte';
	import TransactionsSankey from '$lib/components/TransactionsSankey.svelte';
	import {
		addCategory,
		categories,
		categoriesLoading,
		loadCategories
	} from '$lib/stores/categories';
	import { centsToNumber, formatCurrencyAmount, formatSignedCents } from '$lib/money';
	import { persistedBoolean } from '$lib/stores/persistedBoolean';
	import { user } from '../../user';
	import { t } from 'svelte-i18n';

	type SankeyFilter = {
		kind: 'node' | 'link';
		entryType: 'credit' | 'debit';
		categoryId: number | null;
		label: string;
	};

	let transactions = $state<Transaction[]>([]);
	let summary = $state<TransactionSummary | null>(null);
	let loading = $state(false);
	let listError = $state('');
	let updateError = $state('');
	let categoryUpdates: Record<number, boolean> = $state({});
	let lastUserId: number | null = $state(null);
	let categoryEditorOpen = $state(false);
	let categoryEditorName = $state('');
	let categoryEditorColor: string | null = $state(null);
	let categoryEditorTransactionId: number | null = $state(null);
	let categoryEditorSaving = $state(false);
	let categoryParentMap = $state(new Map<number, number>());

	let fromDate = $state('');
	let toDate = $state('');
	let sourceFileId = $state('');
	let entryType = $state('');
	let searchText = $state('');
	let accountNumber = $state('');
	let cardNumber = $state('');
	let categoryFilter = $state('');
	let calendarRange = $state('');
	let calendarElement: (HTMLElement & { value?: string }) | null = $state(null);
	let calendarOpen = $state(false);
	let calendarPopover: HTMLElement | null = $state(null);
	let calendarInput: HTMLInputElement | null = $state(null);
	let calendarUpdating = $state(false);
	let sankeyFilter: SankeyFilter | null = $state(null);
	let textFilterDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	let lastAppliedSignature = '';
	let lastAppliedTextSignature = '';
	let lastAppliedNonTextSignature = '';
	const textFilterDebounceMs = 400;
	let textFilterSignature = $state('');
	let nonTextFilterSignature = $state('');
	const advancedFiltersOpen = persistedBoolean('transactions.advancedFilters.open', false);

	let assignableCategories = $derived($categories.filter((category) => !category.isGroup));
	$effect(() => {
		categoryParentMap = new Map($categories.map((category) => [category.id, category.parentId]));
	});

	function formatYmd(date: Date): string {
		const year = date.getFullYear();
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	function formatDate(value: string): string {
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) {
			return value;
		}
		const year = date.getFullYear();
		const month = String(date.getMonth() + 1).padStart(2, '0');
		const day = String(date.getDate()).padStart(2, '0');
		return `${year}-${month}-${day}`;
	}

	function handleCalendarRangeChange(event: Event) {
		if (calendarUpdating) {
			return;
		}
		const target = event.currentTarget as HTMLInputElement;
		const value = target.value || '';
		if (!value) {
			fromDate = '';
			toDate = '';
			calendarOpen = false;
			return;
		}
		const [start, end] = value.split('/');
		fromDate = start ?? '';
		toDate = end ?? '';
		if (fromDate && toDate) {
			calendarOpen = false;
		}
	}

	function toggleCalendar() {
		calendarOpen = !calendarOpen;
	}

	function closeCalendar() {
		calendarOpen = false;
	}

	function applyPresetRange(label: string) {
		const today = new Date();
		today.setHours(0, 0, 0, 0);

		let start: Date;
		let end: Date;

		switch (label) {
			case 'current-month': {
				const year = today.getFullYear();
				const month = today.getMonth();
				start = new Date(year, month, 1);
				end = new Date(year, month + 1, 0);
				break;
			}
			case 'previous-month': {
				const year = today.getFullYear();
				const month = today.getMonth() - 1;
				start = new Date(year, month, 1);
				end = new Date(year, month + 1, 0);
				break;
			}
			case 'current-year': {
				const year = today.getFullYear();
				start = new Date(year, 0, 1);
				end = new Date(year, 11, 31);
				break;
			}
			case 'previous-year': {
				const year = today.getFullYear() - 1;
				start = new Date(year, 0, 1);
				end = new Date(year, 11, 31);
				break;
			}
			case 'last-30-days': {
				end = new Date(today);
				start = new Date(today);
				start.setDate(start.getDate() - 29);
				break;
			}
			case 'last-90-days': {
				end = new Date(today);
				start = new Date(today);
				start.setDate(start.getDate() - 89);
				break;
			}
			default:
				return;
		}

		calendarUpdating = true;
		fromDate = formatYmd(start);
		toDate = formatYmd(end);
		calendarOpen = false;
		queueMicrotask(() => {
			calendarUpdating = false;
		});
	}

	function formatSummaryDateRange(start: string, end: string): string {
		if (!start && !end) {
			return '—';
		}
		if (start && end) {
			return `${start} — ${end}`;
		}
		return start || end;
	}

	function clearTextFilterDebounce() {
		if (textFilterDebounceTimer) {
			clearTimeout(textFilterDebounceTimer);
			textFilterDebounceTimer = null;
		}
	}

	function triggerFilterLoad() {
		const signature = `${nonTextFilterSignature}|${textFilterSignature}`;
		if (signature === lastAppliedSignature) {
			return;
		}
		lastAppliedSignature = signature;
		lastAppliedTextSignature = textFilterSignature;
		lastAppliedNonTextSignature = nonTextFilterSignature;
		void loadTransactions();
	}

	function scheduleTextFilterLoad() {
		clearTextFilterDebounce();
		const expectedSignature = textFilterSignature;
		textFilterDebounceTimer = setTimeout(() => {
			textFilterDebounceTimer = null;
			if (expectedSignature !== textFilterSignature) {
				return;
			}
			triggerFilterLoad();
		}, textFilterDebounceMs);
	}

	async function loadTransactions() {
		if (!$user || !$user.id) {
			transactions = [];
			summary = null;
			listError = '';
			loading = false;
			return;
		}

		loading = true;
		listError = '';

		try {
			const response = await Transactions.listTransactions({
				fromDate,
				toDate,
				sourceFileId: sourceFileId ? Number(sourceFileId) : 0,
				entryType,
				searchText,
				categoryId: categoryFilter ? Number(categoryFilter) : 0,
				accountNumber,
				cardNumber
			});
			transactions = response.items ?? [];
			summary = response.summary ?? null;
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				listError = $t('rules.loginRequired');
				transactions = [];
				summary = null;
				return;
			}
			listError = $t('transactions.errorList');
			summary = null;
		} finally {
			loading = false;
		}
	}

	async function updateTransactionCategory(transactionId: number, categoryId: number | null) {
		updateError = '';
		categoryUpdates = { ...categoryUpdates, [transactionId]: true };
		try {
			const request: { transactionId: number; categoryId?: number } = { transactionId };
			if (categoryId !== null) {
				request.categoryId = categoryId;
			}
			await Transactions.updateTransactionCategory(request);
			transactions = transactions.map((tx) =>
				tx.id === transactionId ? { ...tx, categoryId: categoryId ?? undefined } : tx
			);
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				updateError = $t('rules.loginRequired');
				return;
			}
			updateError = $t('categories.errorAction');
		} finally {
			const { [transactionId]: _removed, ...rest } = categoryUpdates;
			categoryUpdates = rest;
		}
	}

	function closeCategoryMenu(event: MouseEvent) {
		const target = event.currentTarget as HTMLElement | null;
		if (!target) {
			return;
		}
		requestAnimationFrame(() => {
			target.blur();
			const dropdown = target.closest('.dropdown');
			const toggle = dropdown?.querySelector('button');
			if (toggle instanceof HTMLElement) {
				toggle.blur();
			}
		});
	}

	function handleCategorySelect(
		event: MouseEvent,
		transactionId: number,
		categoryId: number | null
	) {
		closeCategoryMenu(event);
		void updateTransactionCategory(transactionId, categoryId);
	}

	function openNewCategoryEditor(event: MouseEvent, transactionId: number) {
		closeCategoryMenu(event);
		categoryEditorTransactionId = transactionId;
		categoryEditorName = '';
		categoryEditorColor = null;
		categoryEditorOpen = true;
	}

	function cancelNewCategoryEditor() {
		categoryEditorOpen = false;
		categoryEditorTransactionId = null;
		categoryEditorName = '';
		categoryEditorColor = null;
		categoryEditorSaving = false;
	}

	async function saveNewCategory() {
		if (categoryEditorSaving) {
			return;
		}
		updateError = '';
		const name = categoryEditorName.trim();
		if (!name) {
			return;
		}
		const color = (categoryEditorColor ?? '').trim();
		categoryEditorSaving = true;
		try {
			const response = await Categories.createCategory({
				name,
				color,
				parentId: 0,
				isGroup: false
			});
			if (!response.category) {
				updateError = $t('categories.errorAction');
				return;
			}
			addCategory(response.category);
			const transactionId = categoryEditorTransactionId;
			cancelNewCategoryEditor();
			if (transactionId !== null) {
				await updateTransactionCategory(transactionId, response.category.id);
			}
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				updateError = $t('rules.loginRequired');
				return;
			}
			updateError = $t('categories.errorAction');
		} finally {
			categoryEditorSaving = false;
		}
	}

	function resetFilters() {
		fromDate = '';
		toDate = '';
		sourceFileId = '';
		entryType = '';
		searchText = '';
		categoryFilter = '';
		accountNumber = '';
		cardNumber = '';
		clearTextFilterDebounce();
		triggerFilterLoad();
	}

	function handleSankeyFilterChange(filter: SankeyFilter | null) {
		sankeyFilter = filter;
	}

	function resetSankeyFilter() {
		sankeyFilter = null;
	}

	function matchesSankeyFilter(transaction: Transaction, filter: SankeyFilter): boolean {
		if (transaction.entryType !== filter.entryType) {
			return false;
		}
		if (filter.categoryId === null) {
			return transaction.categoryId === undefined || transaction.categoryId === null;
		}
		if (transaction.categoryId === undefined || transaction.categoryId === null) {
			return false;
		}
		let current = transaction.categoryId;
		const visited = new Set<number>();
		while (current) {
			if (current === filter.categoryId) {
				return true;
			}
			if (visited.has(current)) {
				break;
			}
			visited.add(current);
			const parentId = categoryParentMap.get(current);
			if (!parentId) {
				break;
			}
			current = parentId;
		}
		return false;
	}

	function resolveSankeyCategoryLabel(filter: SankeyFilter): string {
		if (filter.label) {
			return filter.label;
		}
		if (filter.categoryId === null) {
			return $t('transactions.noCategory');
		}
		return (
			$categories.find((category) => category.id === filter.categoryId)?.name ||
			`${$t('transactions.category')} ${filter.categoryId}`
		);
	}

	function formatSankeyFilterLabel(filter: SankeyFilter): string {
		const entryLabel =
			filter.entryType === 'credit' ? $t('transactions.credit') : $t('transactions.debit');
		const categoryLabel = resolveSankeyCategoryLabel(filter);
		return `${entryLabel}: ${categoryLabel}`;
	}

	function formatSankeyFilterType(filter: SankeyFilter): string {
		return filter.kind === 'link' ? $t('transactions.link') : $t('transactions.node');
	}

	$effect(() => {
		if ($user?.id && $user.id !== lastUserId) {
			lastUserId = $user.id;
			clearTextFilterDebounce();
			lastAppliedSignature = '';
			lastAppliedTextSignature = '';
			lastAppliedNonTextSignature = '';
			triggerFilterLoad();
			void loadCategories();
		}
	});

	$effect(() => {
		if ($user?.id) {
			const nextSignature = `${nonTextFilterSignature}|${textFilterSignature}`;
			if (nextSignature !== lastAppliedSignature) {
				const nonTextChanged = nonTextFilterSignature !== lastAppliedNonTextSignature;
				if (nonTextChanged) {
					clearTextFilterDebounce();
					triggerFilterLoad();
				} else {
					scheduleTextFilterLoad();
				}
			}
		}
	});

	$effect(() => {
		textFilterSignature = [searchText, sourceFileId, accountNumber, cardNumber].join('|');
	});

	$effect(() => {
		nonTextFilterSignature = [fromDate, toDate, entryType, categoryFilter].join('|');
	});

	let tableTransactions = $derived(
		sankeyFilter
			? transactions.filter((tx) => matchesSankeyFilter(tx, sankeyFilter!))
			: transactions
	);

	$effect(() => {
		if (fromDate && toDate) {
			const next = `${fromDate}/${toDate}`;
			if (calendarRange !== next) {
				calendarRange = next;
			}
		} else if (!fromDate && !toDate && calendarRange) {
			calendarRange = '';
		}
	});

	$effect(() => {
		if (!calendarUpdating && calendarElement && calendarElement.value !== calendarRange) {
			calendarElement.value = calendarRange;
		}
	});

	let calendarInputValue = $derived(
		fromDate && toDate
			? `${fromDate} — ${toDate}`
			: fromDate || toDate
				? `${fromDate || '—'} — ${toDate || '—'}`
				: ''
	);

	onMount(() => {
		void import('cally');

		const handleDocumentClick = (event: MouseEvent) => {
			if (!calendarOpen) {
				return;
			}
			const target = event.target as Node | null;
			if (calendarPopover && target && calendarPopover.contains(target)) {
				return;
			}
			if (calendarInput && target && calendarInput.contains(target)) {
				return;
			}
			calendarOpen = false;
		};

		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				calendarOpen = false;
			}
		};

		document.addEventListener('click', handleDocumentClick, true);
		window.addEventListener('keydown', handleKeydown);

		return () => {
			document.removeEventListener('click', handleDocumentClick, true);
			window.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

<svelte:head>
	<title>{$t('transactions.title')}</title>
</svelte:head>

<section class="mx-auto w-full max-w-none">
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body gap-6">
			<div class="space-y-2">
				<h1 class="text-2xl font-semibold">{$t('transactions.title')}</h1>
			</div>

			<div class="grid gap-4 lg:grid-cols-2">
				<div class="form-control">
					<label class="label" for="date-range">
						<span class="label-text">{$t('transactions.dateRange')}</span>
					</label>
					<div class="relative">
						<input
							class="input input-bordered w-full"
							type="text"
							id="date-range"
							readonly
							placeholder={$t('transactions.dateRange')}
							bind:this={calendarInput}
							value={calendarInputValue}
							onclick={toggleCalendar}
						/>
						{#if calendarOpen}
							<div
								class="absolute z-20 mt-2 rounded-box border border-base-200 bg-base-100 p-2 shadow"
								bind:this={calendarPopover}
							>
								<div class="mb-2 grid gap-2 sm:grid-cols-2">
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('current-month')}
									>
										{$t('transactions.presets.currentMonth')}
									</button>
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('previous-month')}
									>
										{$t('transactions.presets.previousMonth')}
									</button>
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('current-year')}
									>
										{$t('transactions.presets.currentYear')}
									</button>
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('previous-year')}
									>
										{$t('transactions.presets.previousYear')}
									</button>
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('last-30-days')}
									>
										{$t('transactions.presets.last30Days')}
									</button>
									<button
										class="btn btn-ghost btn-xs justify-start"
										type="button"
										onclick={() => applyPresetRange('last-90-days')}
									>
										{$t('transactions.presets.last90Days')}
									</button>
								</div>
								<calendar-range
									class="cally"
									bind:this={calendarElement}
									value={calendarRange}
									onchange={handleCalendarRangeChange}
								>
									<calendar-month></calendar-month>
								</calendar-range>
								<div class="mt-2 text-xs opacity-70">
									{fromDate || '—'} — {toDate || '—'}
								</div>
							</div>
						{/if}
					</div>
				</div>
				<div class="form-control flex flex-col">
					<label class="label" for="category-filter">
						<span class="label-text">{$t('transactions.category')}</span>
					</label>
					<select
						class="select select-bordered"
						id="category-filter"
						bind:value={categoryFilter}
						disabled={$categoriesLoading}
					>
						<option value="">{$t('common.actions')}</option>
						{#each assignableCategories as category}
							<option value={String(category.id)}>{category.name}</option>
						{/each}
					</select>
				</div>
				<div class="form-control flex flex-col">
					<label class="label" for="search-text">
						<span class="label-text">{$t('transactions.searchPlaceholder')}</span>
					</label>
					<input
						class="input input-bordered w-full"
						type="text"
						id="search-text"
						bind:value={searchText}
						placeholder={$t('transactions.searchExample')}
					/>
				</div>
				<div class="form-control flex flex-col">
					<label class="label" for="entry-type">
						<span class="label-text">{$t('transactions.entryType')}</span>
					</label>
					<select class="select select-bordered" id="entry-type" bind:value={entryType}>
						<option value="">{$t('common.actions')}</option>
						<option value="debit">{$t('transactions.debit')}</option>
						<option value="credit">{$t('transactions.credit')}</option>
					</select>
				</div>
			</div>

			<details
				class="collapse collapse-arrow border border-base-200 bg-base-100"
				bind:open={$advancedFiltersOpen}
			>
				<summary class="collapse-title text-sm font-medium"
					>{$t('transactions.advancedFilters')}</summary
				>
				<div class="collapse-content">
					<div class="grid gap-4 lg:grid-cols-3">
						<div class="form-control flex flex-col">
							<label class="label" for="source-file-id">
								<span class="label-text">{$t('transactions.sourceFileId')}</span>
							</label>
							<input
								class="input input-bordered"
								type="text"
								id="source-file-id"
								bind:value={sourceFileId}
								placeholder="12"
							/>
						</div>
						<div class="form-control flex flex-col">
							<label class="label" for="account-number">
								<span class="label-text">{$t('transactions.accountNumber')}</span>
							</label>
							<input
								class="input input-bordered"
								type="text"
								id="account-number"
								bind:value={accountNumber}
							/>
						</div>
						<div class="form-control flex flex-col">
							<label class="label" for="card-number">
								<span class="label-text">{$t('transactions.cardNumber')}</span>
							</label>
							<input
								class="input input-bordered"
								type="text"
								id="card-number"
								bind:value={cardNumber}
							/>
						</div>
					</div>
				</div>
			</details>

			<div class="flex flex-wrap justify-end gap-3">
				<button class="btn btn-ghost" type="button" onclick={resetFilters} disabled={loading}>
					{$t('transactions.resetFilters')}
				</button>
			</div>

			{#if listError}
				<div class="alert alert-error">
					<span>{listError}</span>
				</div>
			{/if}
			{#if updateError}
				<div class="alert alert-error">
					<span>{updateError}</span>
				</div>
			{/if}

			{#if loading}
				<div class="text-sm opacity-70">{$t('transactions.loading')}</div>
			{:else if transactions.length === 0}
				<div class="text-sm opacity-70">{$t('transactions.empty')}</div>
			{:else}
				{#if summary}
					<div class="stats stats-vertical lg:stats-horizontal bg-base-100 shadow">
						<div class="stat">
							<div class="stat-title">{$t('transactions.count')}</div>
							<div class="stat-value text-lg">{summary.count}</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('transactions.totalAmount')}</div>
							<div class="stat-value text-lg">
								{formatCurrencyAmount(centsToNumber(summary.total), summary.currency)}
							</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('transactions.averageAmount')}</div>
							<div class="stat-value text-lg">
								{formatCurrencyAmount(centsToNumber(summary.average), summary.currency)}
							</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('transactions.medianAmount')}</div>
							<div class="stat-value text-lg">
								{formatCurrencyAmount(centsToNumber(summary.median), summary.currency)}
							</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('transactions.uniqueAccounts')}</div>
							<div class="stat-value text-lg">{summary.uniqueAccounts}</div>
						</div>
						<div class="stat">
							<div class="stat-title">{$t('transactions.dateRange')}</div>
							<div class="stat-value text-lg">
								{formatSummaryDateRange(summary.dateRangeStart, summary.dateRangeEnd)}
							</div>
						</div>
					</div>
				{/if}
				<TransactionsSankey
					{transactions}
					categories={$categories}
					onfilterChange={handleSankeyFilterChange}
				/>
				{#if sankeyFilter}
					<div
						class="flex flex-wrap items-center justify-between gap-3 rounded-box border border-base-200 bg-base-100 p-3 text-sm"
					>
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-medium">{$t('transactions.sankeyFilter')}:</span>
							<span class="badge badge-outline">{formatSankeyFilterType(sankeyFilter)}</span>
							<span>{formatSankeyFilterLabel(sankeyFilter)}</span>
							<button class="btn btn-ghost btn-xs" type="button" onclick={resetSankeyFilter}>
								{$t('common.reset')}
							</button>
						</div>
					</div>
				{/if}
				{#if tableTransactions.length === 0}
					<div class="text-sm opacity-70">{$t('transactions.empty')}</div>
				{:else}
					<div class="overflow-x-auto">
						<table class="table">
							<thead>
								<tr>
									<th>{$t('transactions.date')}</th>
									<th>{$t('transactions.description')}</th>
									<th>{$t('transactions.category')}</th>
									<th class="text-right">{$t('transactions.amount')}</th>
									<th>{$t('transactions.currency')}</th>
								</tr>
							</thead>
							<tbody>
								{#each tableTransactions as tx}
									<tr>
										<td class="whitespace-nowrap">{formatDate(tx.postedDate)}</td>
										<td>
											<div class="font-medium">{tx.description}</div>
											<div class="text-xs opacity-70">ID: {tx.transactionId || '—'}</div>
										</td>
										<td class="whitespace-nowrap">
											<div class="dropdown dropdown-start">
												<button
													class="p-0 cursor-pointer"
													type="button"
													disabled={$categoriesLoading ||
														!assignableCategories.length ||
														categoryUpdates[tx.id]}
												>
													{#if tx.categoryId}
														<CategoryBadge
															name={$categories.find((category) => category.id === tx.categoryId)
																?.name || $t('transactions.category')}
															color={$categories.find((category) => category.id === tx.categoryId)
																?.color || ''}
														/>
													{:else}
														<CategoryBadge name={$t('transactions.noCategory')} />
													{/if}
												</button>
												<ul
													class="menu dropdown-content z-20 mt-2 w-56 rounded-box bg-base-100 p-2 shadow"
												>
													<li>
														<button
															type="button"
															onclick={(event) => handleCategorySelect(event, tx.id, null)}
														>
															<CategoryBadge name={$t('transactions.noCategory')} />
														</button>
													</li>
													{#each assignableCategories as category}
														<li>
															<button
																type="button"
																onclick={(event) => handleCategorySelect(event, tx.id, category.id)}
															>
																<CategoryBadge name={category.name} color={category.color} />
															</button>
														</li>
													{/each}
													<li class="mt-1 border-t border-base-200 pt-1">
														<button
															type="button"
															onclick={(event) => openNewCategoryEditor(event, tx.id)}
														>
															{$t('categories.modalTitleCreate')}
														</button>
													</li>
												</ul>
											</div>
										</td>
										<td class="text-right font-medium"
											>{formatSignedCents(tx.amount, tx.entryType)}</td
										>
										<td>{tx.currency}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</section>

<CategoryEditorModal
	open={categoryEditorOpen}
	bind:name={categoryEditorName}
	bind:color={categoryEditorColor}
	title={$t('categories.modalTitleCreate')}
	confirmLabel={$t('common.add')}
	saving={categoryEditorSaving}
	showParent={false}
	showGroupToggle={false}
	onsave={saveNewCategory}
	oncancel={cancelNewCategoryEditor}
/>

<style>
</style>
