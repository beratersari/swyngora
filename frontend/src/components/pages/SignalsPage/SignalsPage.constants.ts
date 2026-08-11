import type { CreateScannerRuleArg } from '@/libs/api';

/** Live result poll while the Signals tab is visible. */
export const SIGNALS_RESULTS_POLL_MS = 30_000;

/** Faster poll while a backtest is pending/running. */
export const SIGNALS_BACKTEST_POLL_MS = 4_000;

export const SIGNALS_RESULT_LIMIT = 100;

export const SIGNALS_INTERVAL_FALLBACK = ['15m', '1h', '4h', '1d'] as const;

export const SIGNALS_BACKTEST_RANGES = [
  { key: '30d', days: 30 },
  { key: '90d', days: 90 },
  { key: '180d', days: 180 },
  { key: '365d', days: 365 },
] as const;

/** Expert long-side swing stack: trend + momentum + volume on 4h. */
export const SWING_STACK_INTERVAL = '4h';

export const SWING_STACK_RULES: CreateScannerRuleArg[] = [
  {
    type: 'rsi',
    interval: SWING_STACK_INTERVAL,
    rsiPeriod: 14,
    rsiCondition: 'below',
    rsiThreshold: 40,
  },
  {
    type: 'ma_crossover',
    interval: SWING_STACK_INTERVAL,
    maFastPeriod: 12,
    maSlowPeriod: 26,
    maDirection: 'golden_cross',
  },
  {
    type: 'volume_increase',
    interval: SWING_STACK_INTERVAL,
    volumeLookback: 20,
    volumeMinRatio: 2,
  },
];
