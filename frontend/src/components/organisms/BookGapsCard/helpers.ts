export function money(value: string | undefined): string {
  const n = Number(value);
  if (!value || Number.isNaN(n)) return '—';
  return n.toLocaleString('en-US', { maximumFractionDigits: 0 });
}

export function pct(value: number | undefined): string {
  if (value === undefined || Number.isNaN(value)) return '—';
  return `${value.toFixed(2)}%`;
}
