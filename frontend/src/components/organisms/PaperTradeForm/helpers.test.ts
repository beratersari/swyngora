import { describe, expect, it } from 'vitest';
import { kindFromOrderType, toApiOrderType, validateTradeForm } from './helpers';

describe('PaperTradeForm helpers', () => {
  it('maps kind to API order type', () => {
    expect(toApiOrderType('limit', 'buy')).toBe('limit_buy');
    expect(toApiOrderType('limit', 'sell')).toBe('limit_sell');
    expect(toApiOrderType('oco', 'sell')).toBe('oco');
    expect(toApiOrderType('market', 'buy')).toBe('market');
  });

  it('maps API type back to kind', () => {
    expect(kindFromOrderType('limit_buy')).toBe('limit');
    expect(kindFromOrderType('stop_loss')).toBe('stop_loss');
  });

  it('validates required fields', () => {
    expect(
      validateTradeForm({
        exchange: 'binance',
        symbol: 'BTCUSDT',
        orderType: 'market',
        side: 'buy',
        quantity: 0.01,
      }),
    ).toBeNull();
    expect(
      validateTradeForm({
        exchange: 'binance',
        symbol: 'BTCUSDT',
        orderType: 'limit_buy',
        side: 'buy',
        quantity: 0.01,
      }),
    ).toBe('triggerPrice');
    expect(
      validateTradeForm({
        exchange: 'binance',
        symbol: 'BTCUSDT',
        orderType: 'oco',
        side: 'sell',
        quantity: 1,
        takeProfitPrice: 120,
        stopLossPrice: 90,
      }),
    ).toBeNull();
  });
});
