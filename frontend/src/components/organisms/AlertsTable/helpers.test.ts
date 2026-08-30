import { describe, expect, it } from 'vitest';
import { alertConditionLabel, alertSymbolLabel } from './helpers';

describe('AlertsTable helpers', () => {
  it('labels feed and cascade alerts', () => {
    expect(alertConditionLabel({ kind: 'liquidation_feed', targetPrice: 300 })).toBe(
      'feed down ≥ 300s',
    );
    expect(alertConditionLabel({ kind: 'liquidation_cascade', condition: 'extreme' })).toBe(
      'cascade ≥ extreme',
    );
    expect(alertSymbolLabel({ kind: 'liquidation_feed', symbol: 'ALL' })).toBe('all coins');
    expect(alertSymbolLabel({ kind: 'liquidation_cascade', symbol: 'BTCUSDT' })).toBe('BTCUSDT');
  });
});
