import { describe, expect, it } from 'vitest';
import { marginBracketChange, marginCloseQuantity } from './PortfolioMarginPositionsTable.helpers';

const long = {
  side: 'long' as const,
  entryPrice: 100,
  openedStop: 90,
  openedTakeProfit: 120,
};

describe('marginBracketChange', () => {
  it('sends only a changed stop', () => {
    expect(marginBracketChange({ ...long, stopLoss: 80, takeProfit: 120 })).toEqual({
      change: { stopLoss: 80 },
    });
  });

  it('sends only a changed take profit on a short', () => {
    expect(
      marginBracketChange({
        side: 'short',
        entryPrice: 100,
        openedStop: 110,
        openedTakeProfit: 80,
        stopLoss: 110,
        takeProfit: 70,
      }),
    ).toEqual({ change: { takeProfit: 70 } });
  });

  it('clears a stop that was emptied and leaves the take profit', () => {
    expect(marginBracketChange({ ...long, stopLoss: null, takeProfit: 120 })).toEqual({
      change: { clearStopLoss: true },
    });
  });

  it('clears both when both prices are emptied', () => {
    expect(marginBracketChange({ ...long, stopLoss: null, takeProfit: null })).toEqual({
      change: { clearStopLoss: true, clearTakeProfit: true },
    });
  });

  it('adds a stop that the position did not have', () => {
    expect(
      marginBracketChange({
        ...long,
        openedStop: null,
        openedTakeProfit: null,
        stopLoss: 90,
        takeProfit: null,
      }),
    ).toEqual({ change: { stopLoss: 90 } });
  });

  it('reports unchanged when the prices match', () => {
    expect(marginBracketChange({ ...long, stopLoss: 90, takeProfit: 120 })).toEqual({
      issue: 'unchanged',
    });
  });

  it('rejects a long stop at or above the entry', () => {
    expect(marginBracketChange({ ...long, stopLoss: 100, takeProfit: 120 })).toEqual({
      issue: 'stop',
    });
    expect(marginBracketChange({ ...long, stopLoss: 0, takeProfit: 120 })).toEqual({
      issue: 'stop',
    });
  });

  it('closes part of a 10 BTC position and keeps the rest', () => {
    expect(marginCloseQuantity(10, 4)).toEqual({ full: false, quantity: 4, remain: 6 });
  });

  it('treats the full open size as a full close', () => {
    expect(marginCloseQuantity(10, 10)).toEqual({ full: true });
  });

  it('rejects a close above the open size or a remainder too small to keep', () => {
    expect(marginCloseQuantity(10, 11)).toEqual({ issue: 'quantity' });
    expect(marginCloseQuantity(10, 0)).toEqual({ issue: 'quantity' });
    expect(marginCloseQuantity(10, 10 - 1e-9)).toEqual({ issue: 'quantity' });
  });

  it('rejects a short take profit at or above the entry', () => {
    expect(
      marginBracketChange({
        side: 'short',
        entryPrice: 100,
        openedStop: 110,
        openedTakeProfit: 80,
        stopLoss: 110,
        takeProfit: 100,
      }),
    ).toEqual({ issue: 'takeProfit' });
  });
});
