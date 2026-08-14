/** Stable-enough client idempotency key for one paper trade/cash gesture. */
export function newPaperIdempotencyKey(prefix: string): string {
  const p = (prefix || 'web').replace(/[^a-zA-Z0-9._:-]/g, '').slice(0, 32) || 'web';
  const rand =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
  return `${p}-${rand}`.slice(0, 128);
}
