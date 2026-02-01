<script lang="ts">
	import { onMount } from 'svelte';
	import { Reports } from '$lib/api';
	import type { ReportInfo } from '$lib/gen/api/v1/reports_pb';
	import { Code, ConnectError } from '@connectrpc/connect';
	import { user } from '../../user';
	import { t, date as formatDateI18n } from 'svelte-i18n';

	type UploadStatus = 'idle' | 'uploading' | 'success' | 'error';

	let file = $state<File | null>(null);
	let status = $state<UploadStatus>('idle');
	let errorMessage = $state('');
	let uploadedName = $state('');
	let uploadedSize = $state(0);
	let reports = $state<ReportInfo[]>([]);
	let listError = $state('');
	let loadingReports = $state(false);
	let loadedForUserId = $state<number | null>(null);
	let deletingReportId = $state<number | null>(null);
	let toastMessage = $state('');
	let toastTimeout: ReturnType<typeof setTimeout> | null = null;

	function formatBytes(size: number): string {
		if (size < 1024) {
			return `${size} B`;
		}
		const kb = size / 1024;
		if (kb < 1024) {
			return `${kb.toFixed(1)} KB`;
		}
		const mb = kb / 1024;
		return `${mb.toFixed(1)} MB`;
	}

	function handleFileChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		file = target.files?.[0] ?? null;
		status = 'idle';
		errorMessage = '';
	}

	async function handleUpload() {
		if (!file) {
			status = 'error';
			errorMessage = $t('import.errorNoFile');
			return;
		}

		status = 'uploading';
		errorMessage = '';
		try {
			const buffer = await file.arrayBuffer();
			await Reports.uploadReport({
				filename: file.name,
				data: new Uint8Array(buffer),
				contentType: file.type || 'application/octet-stream'
			});
			uploadedName = file.name;
			uploadedSize = file.size;
			file = null;
			status = 'success';
			showToast($t('import.uploadSuccess'));
			await loadReports();
			return;
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				errorMessage = $t('import.errorLogin');
			} else {
				errorMessage = $t('import.errorUpload');
			}
			status = 'error';
		}
	}

	function formatDate(value: string | Date): string {
		const d = new Date(value);
		if (Number.isNaN(d.getTime())) {
			return String(value);
		}
		return $formatDateI18n(d, {
			year: 'numeric',
			month: 'short',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	async function loadReports() {
		if (!$user || !$user.id) {
			reports = [];
			listError = '';
			loadingReports = false;
			return;
		}
		loadingReports = true;
		listError = '';

		try {
			const response = await Reports.listReports({});
			reports = response.reports ?? [];
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				reports = [];
				listError = $t('import.errorListLogin');
				return;
			}
			listError = $t('import.errorList');
		} finally {
			loadingReports = false;
		}
	}

	async function handleDeleteReport(report: ReportInfo) {
		const confirmed = confirm($t('import.deleteConfirm', { name: report.filename } as any));
		if (!confirmed) {
			return;
		}

		deletingReportId = report.id;
		listError = '';
		try {
			await Reports.deleteReport({ id: report.id });
			await loadReports();
			showToast($t('import.deleteSuccess'));
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				listError = $t('import.errorDeleteLogin');
				return;
			}
			listError = $t('import.errorDelete');
		} finally {
			deletingReportId = null;
		}
	}

	async function handleDownloadReport(report: ReportInfo) {
		listError = '';
		try {
			const response = await Reports.downloadReport({ id: report.id });
			const blob = new Blob([response.data as BlobPart], {
				type: response.contentType || 'application/octet-stream'
			});
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = response.filename || report.filename;
			document.body.appendChild(link);
			link.click();
			link.remove();
			URL.revokeObjectURL(url);
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				listError = $t('import.errorDownloadLogin');
				return;
			}
			listError = $t('import.errorDownload');
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

	$effect(() => {
		if ($user?.id) {
			loadedForUserId = $user.id;
			void loadReports();
		}
	});

	$effect(() => {
		if ($user?.id && $user.id !== loadedForUserId) {
			loadedForUserId = $user.id;
			void loadReports();
		}
	});

	$effect(() => {
		if ($user === undefined) {
			reports = [];
			listError = '';
			loadingReports = false;
		}
	});
</script>

<svelte:head>
	<title>{$t('import.pageTitle')}</title>
</svelte:head>

<section class="mx-auto w-full max-w-2xl">
	{#if toastMessage}
		<div class="toast toast-top toast-end z-50">
			<div class="alert alert-success">
				<span>{toastMessage}</span>
			</div>
		</div>
	{/if}
	<div class="card bg-base-100 shadow-xl">
		<div class="card-body gap-6">
			<div class="space-y-2">
				<h1 class="text-2xl font-semibold">{$t('import.title')}</h1>
				<p class="text-sm opacity-70">
					{$t('import.description')}
				</p>
			</div>

			{#if $user === undefined}
				<div class="alert alert-warning">
					<span>{$t('import.loginRequired')}</span>
				</div>
			{/if}

			<div class="form-control gap-3">
				<label class="label" for="csv-file-input">
					<span class="label-text">{$t('import.fileLabel')}</span>
				</label>
				<input
					class="file-input file-input-bordered w-full"
					type="file"
					id="csv-file-input"
					accept=".csv,text/csv"
					onchange={handleFileChange}
				/>
			</div>

			<div class="flex flex-wrap items-center gap-3">
				<button
					class="btn btn-primary"
					type="button"
					onclick={handleUpload}
					disabled={status === 'uploading' || !file}
				>
					{status === 'uploading' ? $t('import.buttonUploading') : $t('import.button')}
				</button>
				{#if file}
					<span class="text-sm opacity-70">{file.name} · {formatBytes(file.size)}</span>
				{/if}
			</div>

			{#if status === 'error'}
				<div class="alert alert-error">
					<span>{errorMessage}</span>
				</div>
			{/if}

			<div class="divider"></div>

			<div class="space-y-3">
				<h2 class="text-lg font-semibold">{$t('import.uploadedFiles')}</h2>
				{#if loadingReports}
					<div class="text-sm opacity-70">{$t('import.loadingList')}</div>
				{:else if listError}
					<div class="alert alert-error">
						<span>{listError}</span>
					</div>
				{:else if reports.length === 0}
					<div class="text-sm opacity-70">{$t('import.listEmpty')}</div>
				{:else}
					<div class="overflow-x-auto">
						<table class="table">
							<thead>
								<tr>
									<th>{$t('import.table.file')}</th>
									<th>{$t('import.table.date')}</th>
									<th>{$t('import.table.size')}</th>
									<th>{$t('import.table.status')}</th>
									<th></th>
								</tr>
							</thead>
							<tbody>
								{#each reports as report}
									<tr>
										<td>
											<button
												class="link link-primary"
												type="button"
												onclick={() => handleDownloadReport(report)}
											>
												{report.filename}
											</button>
										</td>
										<td>{formatDate(report.uploadedAt)}</td>
										<td>{formatBytes(report.sizeBytes)}</td>
										<td>
											{#if report.statusDescription}
												<span class="tooltip tooltip-left" data-tip={report.statusDescription}>
													<span
														class:text-error={report.status === 'failed'}
														class:font-medium={report.status === 'processed'}
													>
														{report.status}
													</span>
												</span>
											{:else}
												<span class:font-medium={report.status === 'processed'}
													>{report.status}</span
												>
											{/if}
										</td>
										<td class="text-right">
											<div class="dropdown dropdown-left">
												<button
													class="btn btn-ghost btn-xs"
													type="button"
													disabled={deletingReportId === report.id}
												>
													⋮
												</button>
												<ul class="menu dropdown-content rounded-box bg-base-100 p-2 shadow">
													<li>
														<button type="button" onclick={() => handleDeleteReport(report)}>
															{$t('common.delete')}
														</button>
													</li>
												</ul>
											</div>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>
	</div>
</section>
