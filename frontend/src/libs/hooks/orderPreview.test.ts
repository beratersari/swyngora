import { describe, expect, it } from 'vitest';
import { previewDraftReady, toPreviewArg, type OrderPreviewDraft } from './orderPreview';

const spot: OrderPreviewDraft = {
  product: 'spot',
  exchange: 'binance',
  symbol: 'btcusdt',
  type: 'market',
  side: 'buy',
  quantity: 1,
};

describe('previewDraftReady', () => {
  it('prices a market buy once the size is set', () => {
    expect(previewDraftReady(spot)).toBe(true);
    expect(previewDraftReady({ ...spot, quantity: 0 })).toBe(false);
    expect(previewDraftReady({ ...spot, symbol: '  ' })).toBe(false);
  });

  it('waits for the trigger on a limit and both prices on an OCO', () => {
    expect(previewDraftReady({ ...spot, type: 'limit_buy' })).toBe(false);
    expect(previewDraftReady({ ...spot, type: 'limit_buy', triggerPrice: 90 })).toBe(true);
    expect(previewDraftReady({ ...spot, type: 'oco', side: 'sell' })).toBe(false);
    expect(
      previewDraftReady({ ...spot, type: 'oco', side: 'sell', takeProfitPrice: 110, stopLossPrice: 80 }),
    ).toBe(true);
  });

  it('waits for a margin limit price and a leverage from 1 to 10', () => {
    const margin: OrderPreviewDraft = {
      product: 'margin',
      exchange: 'binance',
      symbol: 'ETHUSDT',
      type: 'market',
      side: 'long',
      quantity: 1,
      leverage: 2,
    };
    expect(previewDraftReady(margin)).toBe(true);
    expect(previewDraftReady({ ...margin, leverage: 11 })).toBe(false);
    expect(previewDraftReady({ ...margin, type: 'limit' })).toBe(false);
    expect(previewDraftReady({ ...margin, type: 'limit', limitPrice: 100 })).toBe(true);
  });
});

describe('toPreviewArg', () => {
  it('uppercases the symbol and leaves unset prices off the request', () => {
    const arg = toPreviewArg(spot, 'book-1');
    expect(arg.symbol).toBe('BTCUSDT');
    expect(arg.portfolioId).toBe('book-1');
    expect(arg.product).toBe('spot');
    expect(arg.triggerPrice).toBeUndefined();
    expect(arg.limitPrice).toBeUndefined();
  });
});
