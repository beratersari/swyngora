import { situationChangeLines } from '@/components/molecules/CoinSituation';
import type { PriceAlert } from '@/libs/api/endpoints/alertsApi';
import { formatDateTime } from '@/libs/utils/formatMarket';

type Translate = (key: string, options?: Record<string, unknown>) => string;

/** Watch sentence and the last fire. A past fire does not say the live read still matches. */
export function situationAlertNotes(row: PriceAlert, t: Translate, locale?: string): string[] {
  if ((row.kind ?? 'price') !== 'situation') return [];
  const lines: string[] = [];
  if (row.situationWatch === 'blank') {
    lines.push(t('alerts:situationWatchBlank'));
  } else if (row.situationWatch === 'flat') {
    lines.push(t('alerts:situationWatchFlat'));
  }
  const comparison = row.situationFire?.comparison;
  if (!comparison || comparison.same === true) return lines;
  lines.push(t('alerts:situationFireAt', { time: formatDateTime(row.triggeredAt, locale) }));
  lines.push(
    ...situationChangeLines(row.situationFire?.entry, row.situationFire?.live, comparison, 'live', t),
  );
  return lines;
}

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
  if (kind === 'situation') return 'leaves the entry read';
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
