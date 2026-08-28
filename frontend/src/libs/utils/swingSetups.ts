import type { ScannerResult, ScannerRule, ScannerRuleType } from '@/libs/api';

/** Rolling window for client-side confluence grouping. */
export const SIGNALS_CONFLUENCE_WINDOW_MS = 24 * 60 * 60 * 1000;

export type SwingGrade = 'A' | 'B' | 'C';

export type SwingSetup = {
  key: string;
  exchange: string;
  symbol: string;
  interval: string;
  factors: ScannerRuleType[];
  score: number;
  grade: SwingGrade;
  sameBar: boolean;
  latestAt: string;
  summaries: string[];
  hits: ScannerResult[];
};

export function gradeFromScore(score: number): SwingGrade {
  if (score >= 3) return 'A';
  if (score >= 2) return 'B';
  return 'C';
}

const ATOMIC_TYPES: ScannerRuleType[] = ['rsi', 'ma_crossover', 'volume_increase'];

export function ruleConditions(rule: Pick<ScannerRule, 'type' | 'conditions'>): ScannerRuleType[] {
  if (rule.conditions?.length) {
    return rule.conditions.filter((c): c is ScannerRuleType => ATOMIC_TYPES.includes(c));
  }
  if (rule.type && ATOMIC_TYPES.includes(rule.type)) {
    return [rule.type];
  }
  return [];
}

export function ruleTypeShort(type: ScannerRuleType | string | undefined): string {
  switch (type) {
    case 'rsi':
      return 'RSI';
    case 'ma_crossover':
      return 'EMA';
    case 'volume_increase':
      return 'VOL';
    case 'combo':
      return 'COMBO';
    default:
      return type?.toUpperCase() || '—';
  }
}

/** Compact label for a rule's selected conditions (e.g. RSI+VOL). */
export function ruleFactorsShort(rule: Pick<ScannerRule, 'type' | 'conditions'>): string {
  const conds = ruleConditions(rule);
  if (!conds.length) {
    return ruleTypeShort(rule.type);
  }
  return conds.map((c) => ruleTypeShort(c)).join('+');
}

function describeCondition(rule: ScannerRule, type: ScannerRuleType): string {
  switch (type) {
    case 'rsi':
      return `RSI(${rule.rsiPeriod ?? 14}) ${rule.rsiCondition ?? 'below'} ${rule.rsiThreshold ?? ''}`;
    case 'ma_crossover':
      return `EMA(${rule.maFastPeriod ?? 12}/${rule.maSlowPeriod ?? 26}) ${rule.maDirection ?? 'golden_cross'}`;
    case 'volume_increase':
      return `Volume ≥ ${rule.volumeMinRatio ?? 2}× / ${rule.volumeLookback ?? 20}`;
    default:
      return type;
  }
}

export function describeRule(rule: ScannerRule): string {
  const conds = ruleConditions(rule);
  if (!conds.length) {
    return rule.type;
  }
  const parts = conds.map((c) => describeCondition(rule, c));
  if (parts.length === 1) {
    return parts[0] ?? rule.type;
  }
  const sep = rule.matchMode === 'any' ? ' or ' : ' and ';
  return parts.join(sep);
}

function uniqueTypes(hits: readonly ScannerResult[]): ScannerRuleType[] {
  const set = new Set<ScannerRuleType>();
  for (const h of hits) {
    if (h.ruleType === 'rsi' || h.ruleType === 'ma_crossover' || h.ruleType === 'volume_increase') {
      set.add(h.ruleType);
    }
  }
  const order: ScannerRuleType[] = ['ma_crossover', 'rsi', 'volume_increase'];
  return order.filter((t) => set.has(t));
}

function latestIso(hits: readonly ScannerResult[]): string {
  let best = '';
  let bestMs = -Infinity;
  for (const h of hits) {
    const ms = Date.parse(h.matchedAt || h.marketDataKey || '');
    if (Number.isFinite(ms) && ms >= bestMs) {
      bestMs = ms;
      best = h.matchedAt || h.marketDataKey || '';
    }
  }
  return best;
}

function hasSameBarConfluence(hits: readonly ScannerResult[]): boolean {
  const byBar = new Map<string, Set<string>>();
  for (const h of hits) {
    const key = h.marketDataKey || h.matchedAt || h.id;
    const set = byBar.get(key) ?? new Set<string>();
    set.add(h.ruleType);
    byBar.set(key, set);
    if (set.size >= 2) return true;
  }
  return false;
}

/**
 * Group live scanner hits into swing setups.
 * Score = distinct factor types (trend / momentum / volume) in the window.
 * Grade A = 3/3, B = 2/3. Single-factor hits stay out of the setup grid.
 */
export function buildSwingSetups(
  results: readonly ScannerResult[] | undefined,
  nowMs = Date.now(),
  windowMs = SIGNALS_CONFLUENCE_WINDOW_MS,
): SwingSetup[] {
  if (!results?.length) return [];
  const cutoff = nowMs - windowMs;
  const recent = results.filter((r) => {
    const ms = Date.parse(r.matchedAt || r.marketDataKey || '');
    if (!Number.isFinite(ms)) return true;
    return ms >= cutoff;
  });

  const groups = new Map<string, ScannerResult[]>();
  for (const r of recent) {
    const key = `${r.exchange}|${r.symbol}|${r.interval}`;
    const list = groups.get(key) ?? [];
    list.push(r);
    groups.set(key, list);
  }

  const out: SwingSetup[] = [];
  for (const [key, hits] of groups) {
    const factors = uniqueTypes(hits);
    const score = factors.length;
    if (score < 2) continue;
    const [exchange = '', symbol = '', interval = ''] = key.split('|');
    out.push({
      key,
      exchange,
      symbol,
      interval,
      factors,
      score,
      grade: gradeFromScore(score),
      sameBar: hasSameBarConfluence(hits),
      latestAt: latestIso(hits),
      summaries: hits.map((h) => h.summary).filter(Boolean).slice(0, 4),
      hits,
    });
  }

  out.sort((a, b) => {
    if (b.score !== a.score) return b.score - a.score;
    if (a.sameBar !== b.sameBar) return a.sameBar ? -1 : 1;
    return Date.parse(b.latestAt || '') - Date.parse(a.latestAt || '');
  });
  return out;
}

export function countHitsSince(
  results: readonly ScannerResult[] | undefined,
  sinceMs: number,
): number {
  if (!results?.length) return 0;
  return results.filter((r) => {
    const ms = Date.parse(r.matchedAt || r.marketDataKey || '');
    return Number.isFinite(ms) && ms >= sinceMs;
  }).length;
}

export function backtestRangeIso(
  days: number,
  end = new Date(),
): { rangeStart: string; rangeEnd: string } {
  const rangeEnd = new Date(end);
  const rangeStart = new Date(rangeEnd.getTime() - days * 24 * 60 * 60 * 1000);
  return {
    rangeStart: rangeStart.toISOString(),
    rangeEnd: rangeEnd.toISOString(),
  };
}
