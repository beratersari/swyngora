import type { PortfolioView } from '@/libs/api';

export type RealtimeSymbolRef = {
  exchange: string;
  symbol: string;
};

export type RealtimePriceTick = {
  type: 'price';
  exchange: string;
  symbol: string;
  lastPrice?: string;
  priceChange?: string;
  priceChangePercent?: string;
  openPrice?: string;
  highPrice?: string;
  lowPrice?: string;
  volume?: string;
  quoteVolume?: string;
  tradeCount?: number;
  openTime?: string;
  closeTime?: string;
  ts?: string;
};

export type RealtimePortfolioEvent = {
  type: 'portfolio';
  reason?: string;
  portfolioId?: string;
  portfolio?: PortfolioView;
  order?: Record<string, unknown>;
  trade?: Record<string, unknown>;
  orders?: Record<string, unknown>[];
};

export type RealtimeHello = {
  type: 'hello';
  protocol?: number;
  clientId?: string;
  path?: string;
};

export type RealtimeAck = {
  type: 'ack';
  op?: string;
  portfolioId?: string;
  symbols?: RealtimeSymbolRef[];
};

export type RealtimeError = {
  type: 'error';
  code?: string;
  message?: string;
  portfolioId?: string;
};

export type RealtimePong = { type: 'pong'; ts?: string };

export type RealtimeMessage =
  | RealtimePriceTick
  | RealtimePortfolioEvent
  | RealtimeHello
  | RealtimeAck
  | RealtimeError
  | RealtimePong
  | { type: string; [key: string]: unknown };
