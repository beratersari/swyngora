import type { ScannerCondition, ScannerRule, ScannerRuleType } from '@/libs/api';

const ATOMIC_TYPES: ScannerCondition[] = ['rsi', 'ma_crossover', 'volume_increase'];

export function ruleConditions(rule: Pick<ScannerRule, 'type' | 'conditions'>): ScannerCondition[] {
  if (rule.conditions?.length) {
    return rule.conditions.filter((c): c is ScannerCondition => ATOMIC_TYPES.includes(c));
  }
  if (rule.type && ATOMIC_TYPES.includes(rule.type as ScannerCondition)) {
    return [rule.type as ScannerCondition];
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
