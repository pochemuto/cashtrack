<script lang="ts">
    import {onMount} from "svelte";
    import {resolveApiUrl} from "$lib/url";
    import {user} from "../../user";

    type CategoryItem = {
        id: number;
        name: string;
        created_at: string;
    };

    type RuleItem = {
        id: number;
        category_id: number;
        description_contains: string;
        created_at: string;
    };

    let categories: CategoryItem[] = [];
    let rules: RuleItem[] = [];
    let loading = false;
    let listError = "";
    let actionError = "";

    let newCategoryName = "";
    let editingCategoryId: number | null = null;
    let editingCategoryName = "";

    let newRuleCategoryId = "";
    let newRuleText = "";
    let editingRuleId: number | null = null;
    let editingRuleCategoryId = "";
    let editingRuleText = "";
    let menuOpen:
        | {type: "category"; id: number; x: number; y: number}
        | {type: "rule"; id: number; x: number; y: number}
        | null = null;
    let menuElement: HTMLUListElement | null = null;
    let menuAnchor: HTMLElement | null = null;
    let lastUserId: number | null = null;

    $: categoryMap = new Map(categories.map((category) => [category.id, category.name]));

    async function loadData() {
        if (!$user || !$user.id) {
            categories = [];
            rules = [];
            listError = "";
            loading = false;
            return;
        }

        loading = true;
        listError = "";

        try {
            const [categoriesResponse, rulesResponse] = await Promise.all([
                fetch(resolveApiUrl("api/categories"), {credentials: "include"}),
                fetch(resolveApiUrl("api/category-rules"), {credentials: "include"}),
            ]);

            if (!categoriesResponse.ok) {
                listError = "Не удалось загрузить категории.";
                categories = [];
            } else {
                categories = (await categoriesResponse.json()) as CategoryItem[];
            }

            if (!rulesResponse.ok) {
                if (!listError) {
                    listError = "Не удалось загрузить правила.";
                }
                rules = [];
            } else {
                const payload = (await rulesResponse.json()) as RuleItem[] | null;
                rules = Array.isArray(payload) ? payload : [];
            }
        } catch {
            listError = "Не удалось загрузить категории.";
        } finally {
            loading = false;
        }
    }

    async function createCategory() {
        actionError = "";
        const name = newCategoryName.trim();
        if (!name) {
            return;
        }

        try {
            const response = await fetch(resolveApiUrl("api/categories"), {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                credentials: "include",
                body: JSON.stringify({name}),
            });
            if (!response.ok) {
                actionError = "Не удалось добавить категорию.";
                return;
            }
            const created = (await response.json()) as CategoryItem;
            categories = [...categories, created].sort((a, b) => a.name.localeCompare(b.name));
            newCategoryName = "";
        } catch {
            actionError = "Не удалось добавить категорию.";
        }
    }

    function startCategoryEdit(category: CategoryItem) {
        editingCategoryId = category.id;
        editingCategoryName = category.name;
        menuOpen = null;
    }

    function startCategoryEditById(categoryId: number) {
        const category = categories.find((item) => item.id === categoryId);
        if (!category) {
            return;
        }
        startCategoryEdit(category);
    }

    function cancelCategoryEdit() {
        editingCategoryId = null;
        editingCategoryName = "";
    }

    async function saveCategory(categoryId: number) {
        actionError = "";
        const name = editingCategoryName.trim();
        if (!name) {
            return;
        }

        try {
            const response = await fetch(resolveApiUrl(`api/categories/${categoryId}`), {
                method: "PATCH",
                headers: {"Content-Type": "application/json"},
                credentials: "include",
                body: JSON.stringify({name}),
            });
            if (!response.ok) {
                actionError = "Не удалось обновить категорию.";
                return;
            }
            categories = categories.map((category) =>
                category.id === categoryId ? {...category, name} : category
            );
            cancelCategoryEdit();
        } catch {
            actionError = "Не удалось обновить категорию.";
        }
    }

    async function deleteCategory(categoryId: number) {
        actionError = "";
        menuOpen = null;
        try {
            const response = await fetch(resolveApiUrl(`api/categories/${categoryId}`), {
                method: "DELETE",
                credentials: "include",
            });
            if (!response.ok) {
                actionError = "Не удалось удалить категорию.";
                return;
            }
            categories = categories.filter((category) => category.id !== categoryId);
            rules = rules.filter((rule) => rule.category_id !== categoryId);
        } catch {
            actionError = "Не удалось удалить категорию.";
        }
    }

    async function createRule() {
        actionError = "";
        const descriptionContains = newRuleText.trim();
        if (!descriptionContains || !newRuleCategoryId) {
            return;
        }

        try {
            const response = await fetch(resolveApiUrl("api/category-rules"), {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                credentials: "include",
                body: JSON.stringify({
                    category_id: Number(newRuleCategoryId),
                    description_contains: descriptionContains,
                }),
            });
            if (!response.ok) {
                actionError = "Не удалось добавить правило.";
                return;
            }
            const created = (await response.json()) as RuleItem;
            rules = [...rules, created];
            newRuleText = "";
            newRuleCategoryId = "";
        } catch {
            actionError = "Не удалось добавить правило.";
        }
    }

    function startRuleEdit(rule: RuleItem) {
        editingRuleId = rule.id;
        editingRuleCategoryId = String(rule.category_id);
        editingRuleText = rule.description_contains;
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
        editingRuleCategoryId = "";
        editingRuleText = "";
    }

    async function saveRule(ruleId: number) {
        actionError = "";
        const descriptionContains = editingRuleText.trim();
        if (!editingRuleCategoryId || !descriptionContains) {
            return;
        }

        try {
            const response = await fetch(resolveApiUrl(`api/category-rules/${ruleId}`), {
                method: "PATCH",
                headers: {"Content-Type": "application/json"},
                credentials: "include",
                body: JSON.stringify({
                    category_id: Number(editingRuleCategoryId),
                    description_contains: descriptionContains,
                }),
            });
            if (!response.ok) {
                actionError = "Не удалось обновить правило.";
                return;
            }
            rules = rules.map((rule) =>
                rule.id === ruleId
                    ? {
                        ...rule,
                        category_id: Number(editingRuleCategoryId),
                        description_contains: descriptionContains,
                    }
                    : rule
            );
            cancelRuleEdit();
        } catch {
            actionError = "Не удалось обновить правило.";
        }
    }

    async function deleteRule(ruleId: number) {
        actionError = "";
        menuOpen = null;
        try {
            const response = await fetch(resolveApiUrl(`api/category-rules/${ruleId}`), {
                method: "DELETE",
                credentials: "include",
            });
            if (!response.ok) {
                actionError = "Не удалось удалить правило.";
                return;
            }
            rules = rules.filter((rule) => rule.id !== ruleId);
        } catch {
            actionError = "Не удалось удалить правило.";
        }
    }

    function openMenu(event: MouseEvent, type: "category" | "rule", id: number) {
        const target = event.currentTarget as HTMLElement;
        const rect = target.getBoundingClientRect();
        menuOpen = {type, id, x: rect.right, y: rect.bottom};
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
            if (event.key === "Escape") {
                menuOpen = null;
            }
        };

        window.addEventListener("click", handleGlobalClick, true);
        window.addEventListener("keydown", handleKeyDown);

        return () => {
            window.removeEventListener("click", handleGlobalClick, true);
            window.removeEventListener("keydown", handleKeyDown);
        };
    });

    $: if ($user?.id && $user.id !== lastUserId) {
        lastUserId = $user.id;
        void loadData();
    }
</script>

<svelte:head>
    <title>Categories</title>
</svelte:head>

<section class="mx-auto w-full max-w-6xl space-y-6">
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h1 class="text-2xl font-semibold">Категории</h1>
                <p class="text-sm opacity-70">
                    Управляйте списком категорий транзакций.
                </p>
            </div>

            <div class="flex flex-wrap gap-3">
                <input
                    class="input input-bordered flex-1 min-w-[240px]"
                    type="text"
                    placeholder="Название категории"
                    bind:value={newCategoryName}
                />
                <button class="btn btn-primary" type="button" on:click={createCategory}>
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
            {:else if categories.length === 0}
                <div class="text-sm opacity-70">Категории пока не добавлены.</div>
            {:else}
                <div class="overflow-x-auto overflow-y-visible">
                    <table class="table">
                        <thead>
                        <tr>
                            <th>Название</th>
                            <th class="text-right">Действия</th>
                        </tr>
                        </thead>
                        <tbody>
                        {#each categories as category}
                            <tr>
                                <td>
                                    {#if editingCategoryId === category.id}
                                        <input
                                            class="input input-bordered input-sm w-full"
                                            type="text"
                                            bind:value={editingCategoryName}
                                        />
                                    {:else}
                                        <div class="font-medium">{category.name}</div>
                                    {/if}
                                </td>
                                <td class="text-right">
                                    {#if editingCategoryId === category.id}
                                        <div class="flex justify-end gap-2">
                                            <button class="btn btn-sm btn-primary" type="button" on:click={() => saveCategory(category.id)}>
                                                Сохранить
                                            </button>
                                            <button class="btn btn-sm btn-ghost" type="button" on:click={cancelCategoryEdit}>
                                                Отмена
                                            </button>
                                        </div>
                                    {:else}
                                        <button class="btn btn-ghost btn-sm" type="button" on:click={(event) => openMenu(event, "category", category.id)}>
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

    <div class="card bg-base-100 shadow-xl">
        <div class="card-body gap-6">
            <div class="space-y-2">
                <h2 class="text-xl font-semibold">Правила категоризации</h2>
                <p class="text-sm opacity-70">
                    Пока поддерживается правило "описание содержит".
                </p>
            </div>

            <div class="grid gap-3 lg:grid-cols-[minmax(200px,1fr)_minmax(240px,2fr)_auto]">
                <select class="select select-bordered" bind:value={newRuleCategoryId}>
                    <option value="" disabled>Категория</option>
                    {#each categories as category}
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
                    on:click={createRule}
                    disabled={!categories.length}
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
                        {#each rules as rule}
                            <tr>
                                <td>
                                    {#if editingRuleId === rule.id}
                                        <select class="select select-bordered select-sm" bind:value={editingRuleCategoryId}>
                                            {#each categories as category}
                                                <option value={category.id}>{category.name}</option>
                                            {/each}
                                        </select>
                                    {:else}
                                        <div class="font-medium">{categoryMap.get(rule.category_id) || "—"}</div>
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
                                        <span>{rule.description_contains}</span>
                                    {/if}
                                </td>
                                <td class="text-right">
                                    {#if editingRuleId === rule.id}
                                        <div class="flex justify-end gap-2">
                                            <button class="btn btn-sm btn-primary" type="button" on:click={() => saveRule(rule.id)}>
                                                Сохранить
                                            </button>
                                            <button class="btn btn-sm btn-ghost" type="button" on:click={cancelRuleEdit}>
                                                Отмена
                                            </button>
                                        </div>
                                    {:else}
                                        <button class="btn btn-ghost btn-sm" type="button" on:click={(event) => openMenu(event, "rule", rule.id)}>
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

{#if menuOpen}
    <ul
        bind:this={menuElement}
        class="menu rounded-box bg-base-100 p-2 shadow z-50 w-36"
        style={`position: fixed; top: ${menuOpen.y}px; left: ${menuOpen.x}px; transform: translate(-100%, 0);`}
    >
        {#if menuOpen.type === "category"}
            <li>
                <button type="button" on:click={() => startCategoryEditById(menuOpen.id)}>
                    Редактировать
                </button>
            </li>
            <li>
                <button type="button" on:click={() => deleteCategory(menuOpen.id)}>
                    Удалить
                </button>
            </li>
        {:else}
            <li>
                <button type="button" on:click={() => startRuleEditById(menuOpen.id)}>
                    Редактировать
                </button>
            </li>
            <li>
                <button type="button" on:click={() => deleteRule(menuOpen.id)}>
                    Удалить
                </button>
            </li>
        {/if}
    </ul>
{/if}
