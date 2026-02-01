<script lang="ts">
	import { onMount } from 'svelte';
	import ColorPicker from 'svelte-awesome-color-picker';
	import CategoryBadge from '$lib/components/CategoryBadge.svelte';

	interface Props {
		hex?: string | null;
		label?: string;
		popoverClass?: string;
		inline?: boolean;
		nullable?: boolean;
	}

	let {
		hex = $bindable(null),
		label = 'Цвет',
		popoverClass = 'rounded-box border border-base-200 bg-base-100 p-3 shadow-xl',
		inline = false,
		nullable = true
	}: Props = $props();

	let open = $state(false);
	let trigger: HTMLButtonElement | null = $state(null);
	let panel: HTMLDivElement | null = $state(null);
	let popoverStyle = $state('');

	function updatePosition() {
		if (inline) {
			return;
		}
		if (!trigger) {
			return;
		}
		const rect = trigger.getBoundingClientRect();
		const popoverWidth = 320;
		const left = Math.min(rect.left, window.innerWidth - popoverWidth - 16);
		const top = rect.bottom + 8;
		popoverStyle = `position: fixed; left: ${Math.max(16, left)}px; top: ${top}px; z-index: 60;`;
	}

	function openPicker() {
		if (inline) {
			return;
		}
		open = true;
		updatePosition();
		requestAnimationFrame(updatePosition);
	}

	function closePicker() {
		if (inline) {
			return;
		}
		open = false;
	}

	function togglePicker() {
		if (inline) {
			return;
		}
		if (open) {
			closePicker();
		} else {
			openPicker();
		}
	}

	function handleDocumentClick(event: MouseEvent) {
		if (inline) {
			return;
		}
		if (!open) {
			return;
		}
		const target = event.target as Node | null;
		if (panel?.contains(target) || trigger?.contains(target)) {
			return;
		}
		closePicker();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (inline) {
			return;
		}
		if (event.key === 'Escape') {
			closePicker();
		}
	}

	$effect(() => {
		if (inline) return;

		const handleReposition = () => {
			if (open) {
				updatePosition();
			}
		};

		document.addEventListener('click', handleDocumentClick, true);
		document.addEventListener('keydown', handleKeydown);
		window.addEventListener('resize', handleReposition);
		window.addEventListener('scroll', handleReposition, true);

		return () => {
			document.removeEventListener('click', handleDocumentClick, true);
			document.removeEventListener('keydown', handleKeydown);
			window.removeEventListener('resize', handleReposition);
			window.removeEventListener('scroll', handleReposition, true);
		};
	});
</script>

{#if inline}
	<div class="max-w-[320px]">
		<ColorPicker bind:hex {label} {nullable} isAlpha={false} isTextInput={false} isDialog={false} />
	</div>
{:else}
	<div class="flex items-center gap-2">
		<button
			class="p-0"
			type="button"
			aria-expanded={open}
			onclick={togglePicker}
			bind:this={trigger}
		>
			<CategoryBadge name={hex ?? label} color={hex ?? ''} className="cursor-pointer" />
		</button>
	</div>

	{#if open}
		<div style={popoverStyle} class="max-w-[320px]">
			<div class={popoverClass} bind:this={panel}>
				<ColorPicker
					bind:hex
					{label}
					{nullable}
					isAlpha={false}
					isTextInput={false}
					isDialog={false}
				/>
			</div>
		</div>
	{/if}
{/if}
