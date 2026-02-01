<script lang="ts">
    import {onMount} from "svelte";
    import {Categories} from "$lib/api";
    import type {Category, CategoryRule} from "$lib/gen/api/v1/categories_pb";
    import {Code, ConnectError} from "@connectrpc/connect";
    import {categories, addCategory, loadCategories, removeCategory, updateCategory} from "$lib/stores/categories";
    import {user} from "../../user";
    import CategoryBadge from "$lib/components/CategoryBadge.svelte";
    import CategoryEditorModal from "$lib/components/CategoryEditorModal.svelte";

    let rules: CategoryRule[] = [];
    let loading = false;
    let listError = "";
    let actionError = "";
    let toastMessage = "";
    let toastTimeout: ReturnType<typeof setTimeout> | null = null;

    let editorOpen = false;
    let editorMode: "create" | "edit" = "create";
    let editorCategoryId: number | null = null;
    let editorName = "";
    let editorColor: string | null = null;

    let newRuleCategoryId = "";
    let newRuleText = "";
    let editingRuleId: number | null = null;
    let editingRuleCategoryId = "";
    let editingRuleText = "";
    let applyRulesToAll = false;
    let applyingRules = false;
    let rulesReordering = false;
    let draggingRuleIndex: number | null = null;
    let dragOverIndex: number | null = null;
    let menuOpen:
        | {type: "category"; id: number; x: number; y: number}
        | {type: "rule"; id: number; x: number; y: number}
        | null = null;
    let menuElement: HTMLUListElement | null = null;
    let menuAnchor: HTMLElement | null = null;
    let lastUserId: number | null = null;

    $: categoryMap = new Map($categories.map((category) => [category.id, category.name]));

    async function loadData() {
        if (!$user || !$user.id) {
            await loadCategories();
            rules = [];
            listError = "";
            loading = false;
            return;
        }

        loading = true;
        listError = "";

        try {
            const [categoriesOk, rulesResponse] = await Promise.all([
                loadCategories(),
                Categories.listCategoryRules({}),
            ]);

            if (!categoriesOk) {
                listError = "Не удалось загрузить категории.";
            }

            rules = rulesResponse.rules ?? [];
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                listError = "Нужен вход для просмотра правил.";
                rules = [];
                return;
            }
            listError = "Не удалось загрузить категории.";
            rules = [];
        } finally {
            loading = false;
        }
    }

    async function createCategory() {
        actionError = "";
        const name = editorName.trim();
        if (!name) {
            return;
        }
        const color = (editorColor ?? "").trim();

        try {
            const response = await Categories.createCategory({name, color});
            if (!response.category) {
                actionError = "Не удалось добавить категорию.";
                return;
            }
            addCategory(response.category);
            cancelCategoryEdit();
        } catch {
            actionError = "Не удалось добавить категорию.";
        }
    }

    function openCategoryEditor(category: Category) {
        editorMode = "edit";
        editorCategoryId = category.id;
        editorName = category.name;
        editorColor = category.color || null;
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
        editorMode = "create";
        editorCategoryId = null;
        editorName = "";
        editorColor = null;
    }

    async function saveCategory(categoryId: number) {
        actionError = "";
        const name = editorName.trim();
        if (!name) {
            return;
        }
        const color = (editorColor ?? "").trim();

        try {
            await Categories.updateCategory({id: categoryId, name, color});
            const existing = $categories.find((category) => category.id === categoryId);
            if (existing) {
                updateCategory({...existing, name, color});
            }
            cancelCategoryEdit();
        } catch {
            actionError = "Не удалось обновить категорию.";
        }
    }

    async function deleteCategory(categoryId: number) {
        actionError = "";
        menuOpen = null;
        try {
            await Categories.deleteCategory({id: categoryId});
            removeCategory(categoryId);
            rules = rules.filter((rule) => rule.categoryId !== categoryId);
        } catch {
            actionError = "Не удалось удалить категорию.";
        }
    }

    function openCreateCategory() {
        editorMode = "create";
        editorCategoryId = null;
        editorName = "";
        editorColor = null;
        editorOpen = true;
    }

    async function handleCategorySave() {
        if (editorMode === "create") {
            await createCategory();
            return;
        }
        if (editorCategoryId !== null) {
            await saveCategory(editorCategoryId);
        }
    }


    async function createRule() {
        actionError = "";
        const descriptionContains = newRuleText.trim();
        if (!descriptionContains || !newRuleCategoryId) {
            return;
        }

        try {
            const response = await Categories.createCategoryRule({
                categoryId: Number(newRuleCategoryId),
                descriptionContains,
            });
            if (!response.rule) {
                actionError = "Не удалось добавить правило.";
                return;
            }
            rules = [...rules, response.rule];
            newRuleText = "";
            newRuleCategoryId = "";
        } catch {
            actionError = "Не удалось добавить правило.";
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
            await Categories.updateCategoryRule({
                id: ruleId,
                categoryId: Number(editingRuleCategoryId),
                descriptionContains,
            });
            rules = rules.map((rule) =>
                rule.id === ruleId
                    ? {
                        ...rule,
                        categoryId: Number(editingRuleCategoryId),
                        descriptionContains,
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
            await Categories.deleteCategoryRule({id: ruleId});
            rules = rules.filter((rule) => rule.id !== ruleId);
        } catch {
            actionError = "Не удалось удалить правило.";
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
            event.dataTransfer.effectAllowed = "move";
            event.dataTransfer.setData("text/plain", String(rules[index]?.id ?? ""));
        }
    }

    function handleRuleDragOver(event: DragEvent, index: number) {
        if (draggingRuleIndex === null) {
            return;
        }
        event.preventDefault();
        dragOverIndex = index;
        if (event.dataTransfer) {
            event.dataTransfer.dropEffect = "move";
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
        actionError = "";
        rulesReordering = true;
        try {
            await Categories.reorderCategoryRules({ruleIds: nextRules.map((rule) => rule.id)});
        } catch {
            actionError = "Не удалось изменить порядок правил.";
            await loadData();
        } finally {
            rulesReordering = false;
        }
    }

    async function applyRules() {
        if (applyingRules) {
            return;
        }
        actionError = "";
        applyingRules = true;
        try {
            const response = await Categories.applyCategoryRules({applyToAll: applyRulesToAll});
            const updatedCount = response.updatedCount ?? 0;
            showToast(`Правила применены. Обновлено транзакций: ${updatedCount}.`);
        } catch (err) {
            if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
                actionError = "Нужен вход для применения правил.";
                return;
            }
            actionError = "Не удалось применить правила.";
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
            toastMessage = "";
            toastTimeout = null;
        }, 3000);
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
                <button class="btn btn-primary" type="button" on:click={openCreateCategory}>
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
                        <div class="flex flex-wrap items-center justify-between gap-3 rounded-box border border-base-200 bg-base-100 p-3">
                            <div class="flex flex-wrap items-center gap-2">
                                <CategoryBadge name={category.name} color={category.color || ""} />
                                {#if category.color}
                                    <span class="text-xs opacity-70">{category.color}</span>
                                {/if}
                            </div>
                            <button
                                class="btn btn-ghost btn-sm"
                                type="button"
                                on:click={(event) => openMenu(event, "category", category.id)}
                            >
                                ⋮
                            </button>
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
                <p class="text-sm opacity-70">
                    Пока поддерживается правило "описание содержит".
                </p>
            </div>

            <div class="flex flex-wrap items-center gap-3">
                <label class="label cursor-pointer gap-2 p-0">
                    <input class="checkbox checkbox-sm" type="checkbox" bind:checked={applyRulesToAll} />
                    <span class="label-text">Применить ко всем транзакциям</span>
                </label>
                <button
                    class="btn btn-outline btn-sm"
                    type="button"
                    on:click={applyRules}
                    disabled={applyingRules || rules.length === 0}
                >
                    {applyingRules ? "Применение..." : "Применить правила"}
                </button>
            </div>

            <div class="grid gap-3 lg:grid-cols-[minmax(200px,1fr)_minmax(240px,2fr)_auto]">
                <select class="select select-bordered" bind:value={newRuleCategoryId}>
                    <option value="" disabled>Категория</option>
                    {#each $categories as category}
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
                    disabled={!$categories.length}
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
                                on:dragover={(event) => handleRuleDragOver(event, index)}
                                on:drop={(event) => handleRuleDrop(event, index)}
                            >
                                <td>
                                    {#if editingRuleId === rule.id}
                                        <select class="select select-bordered select-sm" bind:value={editingRuleCategoryId}>
                                            {#each $categories as category}
                                                <option value={category.id}>{category.name}</option>
                                            {/each}
                                        </select>
                                    {:else}
                                        <div class="flex items-center gap-2">
                                            <button
                                                class="btn btn-ghost btn-xs cursor-grab"
                                                type="button"
                                                draggable={!(rulesReordering || editingRuleId !== null)}
                                                on:dragstart={(event) => handleRuleDragStart(event, index)}
                                                on:dragend={handleRuleDragEnd}
                                                title="Перетащить правило"
                                                aria-label="Перетащить правило"
                                            >
                                                ⋮⋮
                                            </button>
                                            <div class="font-medium">{categoryMap.get(rule.categoryId) || "—"}</div>
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
                                            <button class="btn btn-sm btn-primary" type="button" on:click={() => saveRule(rule.id)}>
                                                Сохранить
                                            </button>
                                            <button class="btn btn-sm btn-ghost" type="button" on:click={cancelRuleEdit}>
                                                Отмена
                                            </button>
                                        </div>
                                    {:else}
                                        <button
                                            class="btn btn-ghost btn-sm"
                                            type="button"
                                            on:click={(event) => openMenu(event, "rule", rule.id)}
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
    title={editorMode === "create" ? "Новая категория" : "Редактировать категорию"}
    confirmLabel={editorMode === "create" ? "Добавить" : "Сохранить"}
    on:save={handleCategorySave}
    on:cancel={cancelCategoryEdit}
/>

{#if menuOpen}
    <ul
        bind:this={menuElement}
        class="menu rounded-box bg-base-100 p-2 shadow z-50 w-36"
        style={`position: fixed; top: ${menuOpen.y}px; left: ${menuOpen.x}px; transform: translate(-100%, 0);`}
    >
        {#if menuOpen.type === "category"}
            <li>
                <button type="button" on:click={() => menuOpen && openCategoryEditorById(menuOpen.id)}>
                    Редактировать
                </button>
            </li>
            <li>
                <button type="button" on:click={() => menuOpen && deleteCategory(menuOpen.id)}>
                    Удалить
                </button>
            </li>
        {:else}
            <li>
                <button type="button" on:click={() => menuOpen && startRuleEditById(menuOpen.id)}>
                    Редактировать
                </button>
            </li>
            <li>
                <button type="button" on:click={() => menuOpen && deleteRule(menuOpen.id)}>
                    Удалить
                </button>
            </li>
        {/if}
    </ul>
{/if}
