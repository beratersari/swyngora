export const DEFAULT_FORENSICS_SYMBOL = 'BTCUSDT';
export const DEFAULT_FORENSICS_EXCHANGE = 'all';
export const DEFAULT_FORENSICS_WINDOW = '1h';
export const DEFAULT_FORENSICS_DURING = '15m';

export const FORENSICS_EXCHANGES = ['all', 'binance', 'bybit'] as const;
export const FORENSICS_WINDOWS = ['15m', '30m', '1h', '2h', '4h'] as const;
export const FORENSICS_DURINGS = ['5m', '15m', '30m', '1h'] as const;
