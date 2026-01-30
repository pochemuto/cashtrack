export function formatCents(value: bigint): string {
    const sign = value < 0n ? "-" : "";
    const absValue = value < 0n ? -value : value;
    const whole = absValue / 100n;
    const fraction = absValue % 100n;
    return `${sign}${whole.toString()}.${fraction.toString().padStart(2, "0")}`;
}

export function formatSignedCents(value: bigint, entryType?: string): string {
    const formatted = formatCents(value);
    if (entryType === "credit" && value > 0n) {
        return `+${formatted}`;
    }
    return formatted;
}

export function centsToNumber(value: bigint): number {
    return Number(value) / 100;
}

export function formatChfAmount(value: number): string {
    const safeValue = Number.isFinite(value) ? value : 0;
    const amount = safeValue < 0 ? 0 : safeValue;
    if (amount >= 100) {
        const grouped = Math.round(amount)
            .toString()
            .replace(/\B(?=(\d{3})+(?!\d))/g, " ");
        return `${grouped} CHF`;
    }
    const [integerPart, fractionalPart] = amount.toFixed(2).split(".");
    const grouped = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, " ");
    return `${grouped}.${fractionalPart} CHF`;
}
