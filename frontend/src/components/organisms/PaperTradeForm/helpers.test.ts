import { describe, expect, it } from 'vitest';
import { canSizeByRisk, kindFromOrderType, toApiOrderType } from './helpers';

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

  it('sizes a buy or a bracket, not a sell', () => {
    expect(canSizeByRisk('market', 'buy')).toBe(true);
    expect(canSizeByRisk('limit', 'buy')).toBe(true);
    expect(canSizeByRisk('bracket', 'sell')).toBe(true);
    expect(canSizeByRisk('market', 'sell')).toBe(false);
    expect(canSizeByRisk('oco', 'sell')).toBe(false);
    expect(canSizeByRisk('stop_loss', 'sell')).toBe(false);
  });
});
