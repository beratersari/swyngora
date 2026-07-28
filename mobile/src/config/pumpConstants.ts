/** Defaults aligned with backend service + MPUMP-A / design. */

export const DEFAULT_PUMP_SCAN_INTERVAL = '15m';
export const DEFAULT_PUMP_SCAN_LOOKBACK_HOURS = 24;
/** Slightly below service default (8) so the tab is less often empty in quiet markets. */
export const DEFAULT_PUMP_SCAN_MIN_RETURN_PCT = 5;
export const DEFAULT_PUMP_SCAN_DIRECTION = 'up' as const;
export const DEFAULT_PUMP_SCAN_MODE = 'close_return' as const;
export const DEFAULT_PUMP_SCAN_SYMBOL_LIMIT = 15;

export const DEFAULT_PUMP_DETAIL_INTERVAL = '1h';
export const DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS = 48;
export const DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT = 5;
export const DEFAULT_PUMP_DETAIL_DIRECTION = 'both' as const;
export const DEFAULT_PUMP_DETAIL_MAX_EVENTS = 10;

export const PUMP_LOOKBACK_OPTIONS = [6, 12, 24, 48] as const;
export const PUMP_THRESHOLD_OPTIONS = [5, 8, 10, 15, 20] as const;
/** Values only — labels from i18n `pumps:directions.*`. */
export const PUMP_DIRECTION_OPTIONS = [
  { value: 'up' },
  { value: 'down' },
  { value: 'both' },
] as const;

export const PUMP_DISCLAIMER =
  'Informational only — not financial advice. Mechanical threshold matching, not a trade signal.';
