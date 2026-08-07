function toFiniteNumber(value: unknown): number | null {
  if (value == null || value === '') return null;
  if (typeof value === 'number') return Number.isFinite(value) ? value : null;
  const n = Number(String(value).replace(/,/g, ''));
  return Number.isFinite(n) ? n : null;
}

/** Compare tick values for desk flash direction. */
export function numericFlashDirection(prev: unknown, next: unknown): 'up' | 'down' | null {
  const a = toFiniteNumber(prev);
  const b = toFiniteNumber(next);
  if (a == null || b == null || a === b) return null;
  return b > a ? 'up' : 'down';
}
