import type { PriceAlert } from '@/libs/api/endpoints/alertsApi';

export function alertConditionLabel(row: PriceAlert): string {
  const kind = row.kind ?? 'price';
  if (kind === 'liquidation_feed') {
    const sec = row.targetPrice && row.targetPrice > 0 ? row.targetPrice : 300;
    return `feed down ≥ ${sec}s`;
  }
  if (kind === 'liquidation_cascade') {
    const grade = row.condition && row.condition !== '' ? row.condition : 'cascade';
    return `cascade ≥ ${grade}`;
  }
  if (kind === 'liquidation_notional') {
    const side = row.condition && row.condition !== '' ? row.condition : 'both';
    const win = row.rangePct && row.rangePct > 0 ? `${row.rangePct}m` : '5m';
    return `${side} ≥ ${row.targetPrice ?? 0} / ${win}`;
  }
  if (kind === 'liquidation_level') {
    const floor = row.condition && row.condition !== '' ? row.condition : 'any';
    const min = row.targetPrice && row.targetPrice > 0 ? ` ≥ ${row.targetPrice}` : '';
    return `level ${floor}${min}`;
  }
  if (kind === 'setup_radar') {
    const min = row.targetPrice && row.targetPrice > 0 ? row.targetPrice : 60;
    const lean = row.condition && row.condition !== '' ? row.condition : 'any';
    return `setup ≥ ${min} (${lean})`;
  }
  if (kind === 'imbalance' || kind === 'wall') {
    return `${kind} ${row.condition ?? ''} ${row.targetPrice ?? ''}`.trim();
  }
  const cmp = row.condition === 'above' ? '≥' : '≤';
  return `${cmp} ${row.targetPrice ?? ''}`;
}

export function alertSymbolLabel(row: PriceAlert): string {
  if (row.kind === 'liquidation_feed') return 'all coins';
  if ((row.symbol ?? '').toUpperCase() === 'ALL') return 'all coins';
  return row.symbol ?? '—';
}
