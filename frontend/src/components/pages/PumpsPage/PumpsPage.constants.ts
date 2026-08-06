export const DEFAULT_PUMP_SCAN_SYMBOL_LIMIT = 20;
export const DEFAULT_PUMP_SCAN_MIN_RETURN_PCT = 5;
export const DEFAULT_PUMP_SCAN_INTERVAL = '15m';
export const PUMP_SCAN_INTERVALS = ['5m', '15m', '1h', '4h'] as const;
export const PUMP_SCAN_QUOTES = ['USDT', 'USD', 'USDC', 'EUR'] as const;
/** Min-return options for the scanner toolbar (matches empty-state guidance). */
export const PUMP_SCAN_MIN_RETURN_OPTIONS = [2, 3, 5, 8, 10, 15] as const;
