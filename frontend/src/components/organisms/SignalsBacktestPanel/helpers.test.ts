import { describe, expect, it } from 'vitest';
import { formatLeverage, hasBacktestHitDetails, tradePnlTone } from './helpers';

describe('SignalsBacktestPanel helpers', () => {
  it('expands rows with trade or factors', () => {
    expect(hasBacktestHitDetails({ id: '1', backtestId: 'b', signalAt: '', closePrice: 1, summary: '' })).toBe(
      false,
    );
    expect(
      hasBacktestHitDetails({
        id: '1',
        backtestId: 'b',
        signalAt: '',
        closePrice: 1,
        summary: '',
        trade: {
          side: 'long',
          leverage: 3,
          entry: 100,
          stopLoss: 97,
          takeProfit: 105,
          outcome: 'tp',
          pnlPct: 15,
        },
      }),
    ).toBe(true);
  });

  it('tones closed PnL and formats leverage', () => {
    expect(tradePnlTone({ side: 'long', leverage: 2, entry: 1, stopLoss: 0.9, takeProfit: 1.2, outcome: 'tp', pnlPct: 8 })).toBe(
      'success',
    );
    expect(tradePnlTone({ side: 'long', leverage: 2, entry: 1, stopLoss: 0.9, takeProfit: 1.2, outcome: 'sl', pnlPct: -6 })).toBe(
      'error',
    );
    expect(tradePnlTone({ side: 'long', leverage: 2, entry: 1, stopLoss: 0.9, takeProfit: 1.2, outcome: 'open', pnlPct: 1 })).toBe(
      'secondary',
    );
    expect(formatLeverage(4)).toBe('4x');
    expect(formatLeverage(0)).toBe('—');
  });
});
