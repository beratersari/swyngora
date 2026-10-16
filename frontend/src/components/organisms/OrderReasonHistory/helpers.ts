export type ReasonHistoryClose = {
  exchange?: string;
  symbol?: string;
  closedAt?: string;
  realizedPnL?: number;
};

export type ReasonHistoryBucket = {
  count?: number;
  profitable?: number;
  realizedPnL?: number;
  closes?: ReasonHistoryClose[];
};

export type ReasonHistoryRow = {
  clause?: string;
  kind?: string;
  still?: ReasonHistoryBucket;
  left?: ReasonHistoryBucket;
};

/** Rows for clauses that are still checked. Server order is kept. */
export function selectedReasonHistory(
  rows: readonly ReasonHistoryRow[] | null | undefined,
  selected: readonly string[],
): ReasonHistoryRow[] {
  if (!rows || rows.length === 0 || selected.length === 0) return [];
  const want = new Set(selected);
  return rows.filter((row) => {
    const clause = row.clause ?? '';
    return clause !== '' && want.has(clause);
  });
}

export function reasonHistoryHasCloses(row: ReasonHistoryRow): boolean {
  return (row.still?.count ?? 0) + (row.left?.count ?? 0) > 0;
}
