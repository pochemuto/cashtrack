<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import "../app.css";

    let {children} = $props();
</script>

<svelte:head>
    <link rel="icon" href={favicon}/>
</svelte:head>


<div class="navbar bg-base-100 shadow-sm">
    <div class="flex-1">
        <a class="btn btn-ghost text-xl">My App</a>
    </div>
    <div class="flex-none gap-2">
        <button class="btn btn-ghost">Settings</button>
        <button class="btn btn-primary">New</button>
    </div>
</div>

<main class="mx-auto w-full max-w-6xl px-4 py-8">
    {@render children()}
    
    <section class="mx-auto w-full max-w-2xl mt-10">
        <div class="card bg-base-100 shadow-xl">
            <figure class="px-6 pt-6">
                <img
                        src="https://images.unsplash.com/photo-1761839258575-038fef381ee7?auto=format&fit=crop&w=1600&q=80"
                        alt="Preview"
                        class="h-56 w-full rounded-xl object-cover"
                />
            </figure>

            <div class="card-body gap-4">
                <div>
                    <h2 class="card-title">Загрузка и поиск</h2>
                    <p class="text-base-content/70">
                        Картинка + поле ввода + кнопка в центральной части.
                    </p>
                </div>

                <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                    <label class="form-control w-full">
                        <div class="label">
                            <span class="label-text">Запрос</span>
                        </div>
                        <input
                                type="text"
                                placeholder="Например: invoice_2025_12"
                                class="input input-bordered w-full"
                        />
                    </label>

                    <button class="btn btn-primary sm:mb-[2px] sm:w-40">
                        Найти
                    </button>
                </div>

                <div class="divider my-0"></div>

                <div class="flex flex-col gap-3 sm:flex-row">
                    <label class="form-control w-full">
                        <div class="label">
                            <span class="label-text">Фото</span>
                        </div>
                        <input type="file" class="file-input file-input-bordered w-full"/>
                    </label>
                    <button class="btn btn-outline sm:mt-8 sm:w-40">
                        Загрузить
                    </button>
                </div>
            </div>
        </div>
    </section>

    <section class="mt-10">
        <div class="flex items-center justify-between gap-3">
            <h3 class="text-lg font-semibold">Таблица</h3>
            <div class="join">
                <input
                        class="input input-bordered join-item w-56"
                        placeholder="Фильтр..."
                        type="text"
                />
                <button class="btn join-item">Применить</button>
            </div>
        </div>

        <div class="mt-4 overflow-x-auto rounded-box border border-base-300 bg-base-100">
            <table class="table">
                <thead>
                <tr>
                    <th>
                        <label>
                            <input type="checkbox" class="checkbox"/>
                        </label>
                    </th>
                    <th>Название</th>
                    <th>Статус</th>
                    <th>Дата</th>
                    <th class="text-right">Действия</th>
                </tr>
                </thead>
                <tbody>
                <tr class="hover">
                    <th>
                        <label>
                            <input type="checkbox" class="checkbox"/>
                        </label>
                    </th>
                    <td>
                        <div class="font-medium">Документ 001</div>
                        <div class="text-sm opacity-60">ID: a1b2c3</div>
                    </td>
                    <td><span class="badge badge-success badge-outline">Ready</span></td>
                    <td class="opacity-80">2025-12-20</td>
                    <td class="text-right">
                        <button class="btn btn-ghost btn-sm">Open</button>
                        <button class="btn btn-ghost btn-sm">Delete</button>
                    </td>
                </tr>

                <tr class="hover">
                    <th>
                        <label>
                            <input type="checkbox" class="checkbox"/>
                        </label>
                    </th>
                    <td>
                        <div class="font-medium">Документ 002</div>
                        <div class="text-sm opacity-60">ID: d4e5f6</div>
                    </td>
                    <td><span class="badge badge-warning badge-outline">Pending</span></td>
                    <td class="opacity-80">2025-12-18</td>
                    <td class="text-right">
                        <button class="btn btn-ghost btn-sm">Open</button>
                        <button class="btn btn-ghost btn-sm">Delete</button>
                    </td>
                </tr>

                <tr class="hover">
                    <th>
                        <label>
                            <input type="checkbox" class="checkbox"/>
                        </label>
                    </th>
                    <td>
                        <div class="font-medium">Документ 003</div>
                        <div class="text-sm opacity-60">ID: z9y8x7</div>
                    </td>
                    <td><span class="badge badge-error badge-outline">Failed</span></td>
                    <td class="opacity-80">2025-12-10</td>
                    <td class="text-right">
                        <button class="btn btn-ghost btn-sm">Open</button>
                        <button class="btn btn-ghost btn-sm">Delete</button>
                    </td>
                </tr>
                </tbody>
                <tfoot>
                <tr>
                    <th></th>
                    <th>Название</th>
                    <th>Статус</th>
                    <th>Дата</th>
                    <th class="text-right">Действия</th>
                </tr>
                </tfoot>
            </table>
        </div>

        <div class="mt-4 flex items-center justify-between">
            <div class="text-sm opacity-70">Показано 3 из 42</div>
            <div class="join">
                <button class="btn join-item">«</button>
                <button class="btn join-item btn-active">1</button>
                <button class="btn join-item">2</button>
                <button class="btn join-item">3</button>
                <button class="btn join-item">»</button>
            </div>
        </div>
    </section>
</main>