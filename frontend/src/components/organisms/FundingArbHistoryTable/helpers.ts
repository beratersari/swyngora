/** UTC date window for a funding-history lookup. `to` is today, `from` is seven days earlier. */
export function defaultFundingHistoryRange(now = new Date()): { from: string; to: string } {
  const to = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  const from = new Date(to);
  from.setUTCDate(from.getUTCDate() - 7);
  const day = (d: Date) => d.toISOString().slice(0, 10);
  return { from: day(from), to: day(to) };
}
