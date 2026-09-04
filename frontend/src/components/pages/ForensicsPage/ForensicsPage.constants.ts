export const DEFAULT_FORENSICS_SYMBOL = 'BTCUSDT';
export const DEFAULT_FORENSICS_EXCHANGE = 'all';
export const DEFAULT_FORENSICS_WINDOW = '1h';
export const DEFAULT_FORENSICS_DURING = '15m';

export const DEFAULT_FORENSICS_VIEW = 'event';
export const DEFAULT_FORENSICS_LOOKBACK = '3d';

export const FORENSICS_EXCHANGES = ['all', 'binance', 'bybit'] as const;
export const FORENSICS_WINDOWS = ['15m', '30m', '1h', '2h', '4h'] as const;
export const FORENSICS_DURINGS = ['5m', '15m', '30m', '1h'] as const;
export const FORENSICS_LOOKBACKS = ['4h', '12h', '24h', '3d', '7d'] as const;
export const FORENSICS_VIEWS = ['event', 'now', 'review'] as const;
export const FORENSICS_SINCES = ['24h', '7d', '30d'] as const;
export const DEFAULT_FORENSICS_SINCE = '7d';
