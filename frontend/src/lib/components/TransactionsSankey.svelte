<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import type { Category } from '$lib/gen/api/v1/categories_pb';
	import type { Transaction } from '$lib/gen/api/v1/transactions_pb';
	import { persistedBoolean } from '$lib/stores/persistedBoolean';
	import { centsToNumber, formatChfAmount } from '$lib/money';

	type SankeyFilter = {
		kind: 'node' | 'link';
		entryType: 'credit' | 'debit';
		categoryId: number | null;
		label: string;
	};

	type SankeyCustomData = [number, string, string, number | null, string, 'node' | 'link'];

	type SankeyChart = {
		data: Array<Record<string, unknown>>;
		layout: Record<string, unknown>;
		config: Record<string, unknown>;
	};

	type CategoryFilterItem = {
		categoryId: number | null;
		label: string;
		color: string;
		count: number;
		amount: number;
	};

	type CategoryDescriptor = {
		key: string;
		label: string;
		color: string;
		categoryId: number | null;
	};

	interface Props {
		transactions?: Transaction[];
		categories?: Category[];
		onfilterChange?: (filter: SankeyFilter | null) => void;
	}

	let { transactions = [], categories = [], onfilterChange }: Props = $props();

	const sankeyOpen = persistedBoolean('transactions.sankey.open', false);

	let plotlyLoading = $state(false);
	let plotlyError = $state('');
	let plotly: any = $state(null);
	let sankeyContainer: HTMLDivElement | null = $state(null);
	let sankeyHandlersAttached = $state(false);
	let lastPlotlyClickAt = 0;
	let plotlyClickHandler: ((event: any) => void) | null = null;
	let containerClickHandler: ((event: MouseEvent) => void) | null = null;
	let categoryVisibility: Record<string, boolean> = $state({});
	let showUnknownCategory = $state(true);
	let visibilityLoaded = $state(false);

	const sankeySourceColor = '#e2e8f0';
	const sankeyCategoryFallbackColor = '#94a3b8';
	const uncategorizedKey = 'null';
	const sankeyVisibilityStorageKey = 'transactions.sankey.categoryVisibility';

	function normalizeColor(value: string | null, fallback: string): string {
		if (!value) {
			return fallback;
		}
		const trimmed = value.trim();
		return trimmed ? trimmed : fallback;
	}

	function colorWithAlpha(value: string, alpha: number, fallback: string): string {
		const trimmed = value.trim();
		if (trimmed.startsWith('#')) {
			const hex = trimmed.slice(1);
			if (hex.length === 3) {
				const r = Number.parseInt(hex[0] + hex[0], 16);
				const g = Number.parseInt(hex[1] + hex[1], 16);
				const b = Number.parseInt(hex[2] + hex[2], 16);
				return `rgba(${r}, ${g}, ${b}, ${alpha})`;
			}
			if (hex.length === 6) {
				const r = Number.parseInt(hex.slice(0, 2), 16);
				const g = Number.parseInt(hex.slice(2, 4), 16);
				const b = Number.parseInt(hex.slice(4, 6), 16);
				return `rgba(${r}, ${g}, ${b}, ${alpha})`;
			}
		}
		if (trimmed.startsWith('rgb(')) {
			const parts = trimmed
				.slice(4, -1)
				.split(',')
				.map((item) => item.trim());
			if (parts.length >= 3) {
				return `rgba(${parts[0]}, ${parts[1]}, ${parts[2]}, ${alpha})`;
			}
		}
		if (trimmed.startsWith('rgba(')) {
			const parts = trimmed
				.slice(5, -1)
				.split(',')
				.map((item) => item.trim());
			if (parts.length >= 3) {
				return `rgba(${parts[0]}, ${parts[1]}, ${parts[2]}, ${alpha})`;
			}
		}
		return fallback;
	}

	function categoryKey(categoryId: number | null): string {
		return categoryId === null ? uncategorizedKey : String(categoryId);
	}

	function parseSankeyFilter(customdata: unknown): SankeyFilter | null {
		if (!Array.isArray(customdata)) {
			return null;
		}
		const entryType = customdata[2];
		const rawCategoryId = customdata[3];
		const label = customdata[4];
		const kind = customdata[5];
		if (entryType !== 'credit' && entryType !== 'debit') {
			return null;
		}
		if (kind !== 'node' && kind !== 'link') {
			return null;
		}
		const categoryId =
			typeof rawCategoryId === 'number'
				? rawCategoryId
				: rawCategoryId === null
					? null
					: Number.isFinite(Number(rawCategoryId))
						? Number(rawCategoryId)
						: null;
		return {
			kind,
			entryType,
			categoryId,
			label: typeof label === 'string' ? label : ''
		};
	}

	function attachSankeyHandlers() {
		if (!sankeyContainer || sankeyHandlersAttached) {
			return;
		}
		const plotlyTarget = sankeyContainer as unknown as {
			on?: (event: string, handler: (event: any) => void) => void;
			removeListener?: (event: string, handler: (event: any) => void) => void;
			removeAllListeners?: (event: string) => void;
		};
		plotlyClickHandler = (event: any) => {
			const now = performance.now();
			const customdata = event?.points?.[0]?.customdata;
			lastPlotlyClickAt = now;
			onfilterChange?.(parseSankeyFilter(customdata));
		};
		plotlyTarget.on?.('plotly_click', plotlyClickHandler);

		containerClickHandler = () => {
			if (!sankeyChart) {
				return;
			}
			if (performance.now() - lastPlotlyClickAt < 250) {
				return;
			}
			onfilterChange?.(null);
		};
		sankeyContainer.addEventListener('click', containerClickHandler);
		sankeyHandlersAttached = true;
	}

	function detachSankeyHandlers() {
		if (!sankeyContainer || !sankeyHandlersAttached) {
			return;
		}
		const plotlyTarget = sankeyContainer as unknown as {
			removeListener?: (event: string, handler: (event: any) => void) => void;
			removeAllListeners?: (event: string) => void;
		};
		if (plotlyClickHandler) {
			if (plotlyTarget.removeListener) {
				plotlyTarget.removeListener('plotly_click', plotlyClickHandler);
			} else if (plotlyTarget.removeAllListeners) {
				plotlyTarget.removeAllListeners('plotly_click');
			}
		}
		if (containerClickHandler) {
			sankeyContainer.removeEventListener('click', containerClickHandler);
		}
		plotlyClickHandler = null;
		containerClickHandler = null;
		sankeyHandlersAttached = false;
	}

	function getCategoryDescriptor(
		categoryId: number | null | undefined,
		categoryLookup: Map<number, Category>
	): CategoryDescriptor {
		if (categoryId === undefined || categoryId === null) {
			return {
				key: 'none',
				label: 'Без категории',
				color: sankeyCategoryFallbackColor,
				categoryId: null
			};
		}
		const category = categoryLookup.get(categoryId);
		if (!category) {
			return {
				key: String(categoryId),
				label: `Категория ${categoryId}`,
				color: sankeyCategoryFallbackColor,
				categoryId
			};
		}
		return {
			key: String(category.id),
			label: category.name || `Категория ${category.id}`,
			color: normalizeColor(category.color, sankeyCategoryFallbackColor),
			categoryId
		};
	}

	function buildCategoryChain(
		categoryId: number,
		categoryLookup: Map<number, Category>
	): CategoryDescriptor[] {
		const chain: CategoryDescriptor[] = [];
		let currentId: number | null = categoryId;
		const visited = new Set<number>();

		while (currentId !== null && currentId !== 0 && !visited.has(currentId)) {
			visited.add(currentId);
			const descriptor = getCategoryDescriptor(currentId, categoryLookup);
			chain.push(descriptor);
			const category = categoryLookup.get(currentId);
			if (!category || !category.parentId) {
				break;
			}
			currentId = category.parentId;
		}

		return chain.reverse();
	}

	function buildCategoryFilterItems(
		items: Transaction[],
		categoryItems: Category[]
	): CategoryFilterItem[] {
		if (!items.length) {
			return [];
		}

		const categoryLookup = new Map<number, Category>();
		for (const category of categoryItems) {
			categoryLookup.set(category.id, category);
		}

		const totals = new Map<string, CategoryFilterItem>();

		for (const tx of items) {
			const amount = centsToNumber(tx.amount);
			if (!Number.isFinite(amount) || amount === 0) {
				continue;
			}
			const value = Math.abs(amount);
			const descriptor = getCategoryDescriptor(tx.categoryId, categoryLookup);
			const key = categoryKey(descriptor.categoryId);
			const existing = totals.get(key);
			if (existing) {
				existing.count += 1;
				existing.amount += value;
			} else {
				totals.set(key, {
					categoryId: descriptor.categoryId,
					label: descriptor.label,
					color: descriptor.color,
					count: 1,
					amount: value
				});
			}
		}

		return Array.from(totals.values()).sort((left, right) => {
			if (right.amount !== left.amount) {
				return right.amount - left.amount;
			}
			return left.label.localeCompare(right.label);
		});
	}

	function syncCategoryVisibility(
		current: Record<string, boolean>,
		items: CategoryFilterItem[]
	): Record<string, boolean> {
		const next: Record<string, boolean> = {};
		let changed = false;

		for (const item of items) {
			const key = categoryKey(item.categoryId);
			const value = current[key] ?? true;
			next[key] = value;
			if (current[key] !== value) {
				changed = true;
			}
		}

		const currentKeys = Object.keys(current);
		if (currentKeys.length !== Object.keys(next).length) {
			changed = true;
		} else if (!changed) {
			for (const key of currentKeys) {
				if (!(key in next)) {
					changed = true;
					break;
				}
			}
		}

		return changed ? next : current;
	}

	function setCategoryVisibility(categoryId: number | null, visible: boolean) {
		categoryVisibility = { ...categoryVisibility, [categoryKey(categoryId)]: visible };
	}

	function showAllCategories() {
		const next: Record<string, boolean> = {};
		for (const item of categoryFilterItems) {
			next[categoryKey(item.categoryId)] = true;
		}
		categoryVisibility = next;
	}

	function hideAllCategories() {
		const next: Record<string, boolean> = {};
		for (const item of categoryFilterItems) {
			next[categoryKey(item.categoryId)] = false;
		}
		categoryVisibility = next;
	}

	function buildSankeyChart(
		items: Transaction[],
		categoryItems: Category[],
		visibility: Record<string, boolean>,
		includeUnknown: boolean
	): SankeyChart | null {
		if (!items.length) {
			return null;
		}

		const categoryLookup = new Map<number, Category>();
		for (const category of categoryItems) {
			categoryLookup.set(category.id, category);
		}

		type FilterMeta = { entryType: 'credit' | 'debit'; categoryId: number | null; label: string };

		const nodeIndex = new Map<string, number>();
		const nodeLabels: string[] = [];
		const nodeColors: string[] = [];
		const links = new Map<
			string,
			{
				source: number;
				target: number;
				value: number;
				color: string;
				count: number;
				meta: FilterMeta | null;
			}
		>();
		const netIncomeLabel = 'Net income';
		const remainderLabel = 'Remainder';
		let totalCredits = 0;
		let totalDebits = 0;
		let totalCreditCount = 0;

		const nodeMetaByIndex: Array<FilterMeta | null> = [];

		const ensureNode = (
			key: string,
			label: string,
			color: string,
			meta: FilterMeta | null = null
		) => {
			if (!nodeIndex.has(key)) {
				nodeIndex.set(key, nodeLabels.length);
				nodeLabels.push(label);
				nodeColors.push(color);
				nodeMetaByIndex.push(meta);
			}
			const index = nodeIndex.get(key) ?? 0;
			if (meta && !nodeMetaByIndex[index]) {
				nodeMetaByIndex[index] = meta;
			}
			return index;
		};

		const nodeStats = new Map<string, { count: number; amount: number }>();

		const addNodeStats = (key: string, value: number, count: number) => {
			const existing = nodeStats.get(key);
			if (existing) {
				existing.amount += value;
				existing.count += count;
			} else {
				nodeStats.set(key, { count, amount: value });
			}
		};

		const addLink = (
			source: number,
			target: number,
			value: number,
			color: string,
			count: number,
			meta: FilterMeta | null
		) => {
			const linkKey = `${source}:${target}`;
			const existing = links.get(linkKey);
			if (existing) {
				existing.value += value;
				existing.count += count;
			} else {
				links.set(linkKey, { source, target, value, color, count, meta });
			}
		};

		const netIncomeIndex = ensureNode('net:income', netIncomeLabel, sankeySourceColor, null);

		const entryNodeKey = (entryType: 'credit' | 'debit', descriptor: CategoryDescriptor) =>
			`${entryType}:${descriptor.key}`;

		const ensureCategoryNode = (entryType: 'credit' | 'debit', descriptor: CategoryDescriptor) =>
			ensureNode(entryNodeKey(entryType, descriptor), descriptor.label, descriptor.color, {
				entryType,
				categoryId: descriptor.categoryId,
				label: descriptor.label
			});

		for (const tx of items) {
			const amount = centsToNumber(tx.amount);
			if (!Number.isFinite(amount) || amount === 0) {
				continue;
			}
			const value = Math.abs(amount);
			const categoryInfo = getCategoryDescriptor(tx.categoryId, categoryLookup);
			const isVisible = visibility[categoryKey(categoryInfo.categoryId)] ?? true;
			if (!isVisible) {
				continue;
			}

			if (tx.entryType === 'credit') {
				totalCredits += value;
				totalCreditCount += 1;
			} else if (tx.entryType === 'debit') {
				totalDebits += value;
			} else {
				continue;
			}

			if (categoryInfo.categoryId === null) {
				const nodeKey = entryNodeKey(tx.entryType, categoryInfo);
				const nodeIndex = ensureCategoryNode(tx.entryType, categoryInfo);
				addNodeStats(nodeKey, value, 1);
				if (tx.entryType === 'credit') {
					addLink(
						nodeIndex,
						netIncomeIndex,
						value,
						colorWithAlpha(categoryInfo.color, 0.45, categoryInfo.color),
						1,
						{ entryType: 'credit', categoryId: null, label: categoryInfo.label }
					);
				} else {
					addLink(netIncomeIndex, nodeIndex, value, categoryInfo.color, 1, {
						entryType: 'debit',
						categoryId: null,
						label: categoryInfo.label
					});
				}
				continue;
			}

			const chain = buildCategoryChain(categoryInfo.categoryId, categoryLookup);
			if (!chain.length) {
				continue;
			}

			for (const entry of chain) {
				addNodeStats(entryNodeKey(tx.entryType, entry), value, 1);
			}

			if (tx.entryType === 'credit') {
				for (let index = chain.length - 1; index >= 0; index -= 1) {
					const current = chain[index];
					const sourceIndex = ensureCategoryNode('credit', current);
					const targetIndex =
						index > 0 ? ensureCategoryNode('credit', chain[index - 1]) : netIncomeIndex;
					addLink(
						sourceIndex,
						targetIndex,
						value,
						colorWithAlpha(current.color, 0.45, current.color),
						1,
						{ entryType: 'credit', categoryId: current.categoryId, label: current.label }
					);
				}
			} else {
				const root = chain[0];
				const rootIndex = ensureCategoryNode('debit', root);
				addLink(netIncomeIndex, rootIndex, value, root.color, 1, {
					entryType: 'debit',
					categoryId: root.categoryId,
					label: root.label
				});
				for (let index = 0; index < chain.length - 1; index += 1) {
					const parent = chain[index];
					const child = chain[index + 1];
					const sourceIndex = ensureCategoryNode('debit', parent);
					const targetIndex = ensureCategoryNode('debit', child);
					addLink(sourceIndex, targetIndex, value, child.color, 1, {
						entryType: 'debit',
						categoryId: child.categoryId,
						label: child.label
					});
				}
			}
		}

		if (!links.size) {
			return null;
		}

		nodeStats.set('net:income', { count: totalCreditCount, amount: totalCredits });

		const remainder = Number((totalCredits - totalDebits).toFixed(2));
		if (includeUnknown && remainder > 0) {
			const remainderIndex = ensureNode(
				`debit:${remainderLabel}`,
				remainderLabel,
				sankeyCategoryFallbackColor,
				null
			);
			addLink(
				netIncomeIndex,
				remainderIndex,
				remainder,
				colorWithAlpha(sankeyCategoryFallbackColor, 0.45, sankeyCategoryFallbackColor),
				0,
				null
			);
			nodeStats.set(`debit:${remainderLabel}`, { count: 0, amount: remainder });
		}

		const sources: number[] = [];
		const targets: number[] = [];
		const values: number[] = [];
		const colors: string[] = [];
		const linkCustomData: SankeyCustomData[] = [];
		const nodeCustomData: SankeyCustomData[] = nodeLabels.map((label, index) => {
			const meta = nodeMetaByIndex[index];
			return [
				0,
				formatChfAmount(0),
				meta?.entryType ?? '',
				meta?.categoryId ?? null,
				meta?.label ?? label,
				'node'
			];
		});

		for (const [key, stats] of nodeStats.entries()) {
			const index = nodeIndex.get(key);
			if (index !== undefined) {
				const meta = nodeMetaByIndex[index];
				nodeCustomData[index] = [
					stats.count,
					formatChfAmount(stats.amount),
					meta?.entryType ?? '',
					meta?.categoryId ?? null,
					meta?.label ?? nodeLabels[index] ?? '',
					'node'
				];
			}
		}

		for (const link of links.values()) {
			sources.push(link.source);
			targets.push(link.target);
			values.push(Number(link.value.toFixed(2)));
			colors.push(link.color);
			linkCustomData.push([
				link.count,
				formatChfAmount(link.value),
				link.meta?.entryType ?? '',
				link.meta?.categoryId ?? null,
				link.meta?.label ?? '',
				'link'
			]);
		}

		const height = Math.min(640, Math.max(280, nodeLabels.length * 24));

		return {
			data: [
				{
					type: 'sankey',
					orientation: 'h',
					node: {
						pad: 18,
						thickness: 16,
						line: { color: 'rgba(0,0,0,0.2)', width: 0.5 },
						label: nodeLabels,
						color: nodeColors,
						customdata: nodeCustomData,
						hovertemplate:
							'%{label}<br>Входящие транзакции: %{customdata[0]}<br>%{customdata[1]}<extra></extra>'
					},
					link: {
						source: sources,
						target: targets,
						value: values,
						color: colors,
						customdata: linkCustomData,
						hovertemplate:
							'%{source.label} → %{target.label}<br>Транзакции: %{customdata[0]}<br>%{customdata[1]}<extra></extra>'
					}
				}
			],
			layout: {
				margin: { l: 10, r: 10, t: 10, b: 10 },
				height,
				paper_bgcolor: 'transparent',
				plot_bgcolor: 'transparent'
			},
			config: {
				displayModeBar: false,
				responsive: true
			}
		};
	}

	function handleSankeyToggle(event: Event) {
		const details = event.currentTarget as HTMLDetailsElement | null;
		if (!details || !details.open || !plotly || !sankeyContainer) {
			return;
		}
		requestAnimationFrame(() => {
			plotly.Plots.resize(sankeyContainer);
		});
	}

	let categoryFilterItems = $derived(buildCategoryFilterItems(transactions, categories));

	$effect(() => {
		if (categoryFilterItems.length > 0) {
			const nextVisibility = syncCategoryVisibility(categoryVisibility, categoryFilterItems);
			// Deep comparison not easily possible here, but `syncCategoryVisibility` logic handles changes
			// However logic relies on object identity or simple key length check.
			// Let's trust `syncCategoryVisibility` returns a new object if changed or same if not.
			if (nextVisibility !== categoryVisibility) {
				categoryVisibility = nextVisibility;
			}
		}
	});

	$effect(() => {
		if (visibilityLoaded && browser) {
			try {
				localStorage.setItem(sankeyVisibilityStorageKey, JSON.stringify(categoryVisibility));
			} catch {
				// Ignore storage failures.
			}
		}
	});

	let sankeyChart = $derived(
		buildSankeyChart(transactions, categories, categoryVisibility, showUnknownCategory)
	);

	$effect(() => {
		if (plotly && sankeyContainer) {
			if (sankeyChart) {
				plotly.react(sankeyContainer, sankeyChart.data, sankeyChart.layout, sankeyChart.config);
			} else {
				plotly.purge(sankeyContainer);
			}
		}
	});

	$effect(() => {
		if (plotly && sankeyContainer && sankeyChart && !sankeyHandlersAttached) {
			attachSankeyHandlers();
		}
	});

	$effect(() => {
		if ((!sankeyContainer || !sankeyChart) && sankeyHandlersAttached) {
			detachSankeyHandlers();
		}
	});

	onMount(() => {
		let cancelled = false;
		plotlyLoading = true;
		import('plotly.js-dist-min')
			.then((module) => {
				if (cancelled) {
					return;
				}
				plotly = module.default ?? module;
			})
			.catch(() => {
				if (!cancelled) {
					plotlyError = 'Не удалось загрузить диаграмму.';
				}
			})
			.finally(() => {
				if (!cancelled) {
					plotlyLoading = false;
				}
			});

		if (browser) {
			try {
				const raw = localStorage.getItem(sankeyVisibilityStorageKey);
				if (raw) {
					const parsed = JSON.parse(raw) as Record<string, unknown> | null;
					if (parsed && typeof parsed === 'object') {
						const restored: Record<string, boolean> = {};
						for (const [key, value] of Object.entries(parsed)) {
							if (typeof value === 'boolean') {
								restored[key] = value;
							}
						}
						if (Object.keys(restored).length > 0) {
							categoryVisibility = restored;
						}
					}
				}
			} catch {
				// Keep default in-memory visibility when storage is unavailable.
			}
		}
		visibilityLoaded = true;

		return () => {
			cancelled = true;
			detachSankeyHandlers();
		};
	});
</script>

<details
	class="collapse collapse-arrow border border-base-200 bg-base-100"
	bind:open={$sankeyOpen}
	ontoggle={handleSankeyToggle}
>
	<summary class="collapse-title text-sm font-medium">Sankey-диаграмма</summary>
	<div class="collapse-content">
		{#if categoryFilterItems.length > 0}
			<div class="mb-4 space-y-3 rounded-box bg-base-100 p-3 text-xs">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<div class="flex flex-wrap gap-2">
						<button class="btn btn-ghost btn-xs" type="button" onclick={showAllCategories}
							>Показать все</button
						>
						<button class="btn btn-ghost btn-xs" type="button" onclick={hideAllCategories}
							>Скрыть все</button
						>
					</div>
				</div>
				<div class="flex flex-nowrap gap-2 overflow-x-auto pb-1">
					{#each categoryFilterItems as item}
						<label class="flex items-center gap-2 rounded-box px-2 py-1 whitespace-nowrap">
							<input
								class="checkbox checkbox-xs"
								type="checkbox"
								checked={categoryVisibility[categoryKey(item.categoryId)] ?? true}
								onchange={(event) =>
									setCategoryVisibility(
										item.categoryId,
										(event.currentTarget as HTMLInputElement).checked
									)}
							/>
							<span class="inline-flex items-center gap-2">
								<span class="h-2.5 w-2.5 rounded-full" style={`background:${item.color};`}></span>
								<span>{item.label}</span>
							</span>
						</label>
					{/each}
					<label class="flex items-center gap-2 rounded-box px-2 py-1 whitespace-nowrap">
						<input
							class="checkbox checkbox-xs"
							type="checkbox"
							bind:checked={showUnknownCategory}
						/>
						<span>Remainder</span>
					</label>
				</div>
			</div>
		{/if}
		{#if plotlyError}
			<div class="text-sm text-error">{plotlyError}</div>
		{:else if plotlyLoading}
			<div class="text-sm opacity-70">Загрузка диаграммы...</div>
		{:else if !sankeyChart}
			<div class="text-sm opacity-70">Недостаточно данных для диаграммы.</div>
		{:else}
			<div class="min-h-[280px] w-full" bind:this={sankeyContainer}></div>
		{/if}
	</div>
</details>
