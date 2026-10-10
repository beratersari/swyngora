import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { isCashEquity, openPositionFor } from './helpers';
import { PositionSituation } from './PositionSituation';

const entry = {
  symbol: 'BTCUSDT',
  priceCvd: { kind: 'price_up_cvd_down' as const },
};

const now = {
  symbol: 'BTCUSDT',
  priceCvd: { kind: 'agrees' as const },
};

describe('PositionSituation', () => {
  it('shows the stored entry read beside the live read', () => {
    renderWithProviders(<PositionSituation entry={entry} now={now} />);
    expect(screen.getByRole('group', { name: 'Position read' })).toBeInTheDocument();
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
  });

  it('keeps the entry line when the live read is still loading', () => {
    renderWithProviders(<PositionSituation entry={entry} nowLoading />);
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Reading the tape now…')).toBeInTheDocument();
    expect(screen.queryByText('Now')).not.toBeInTheDocument();
  });

  it('renders nothing when neither read has a clause', () => {
    renderWithProviders(<PositionSituation />);
    expect(screen.queryByRole('group', { name: 'Position read' })).not.toBeInTheDocument();
  });
});

describe('openPositionFor', () => {
  it('returns the open row for this coin and skips a flat or equity book', () => {
    const row = openPositionFor(
      [
        { exchange: 'binance', symbol: 'ETHUSDT', quantity: 2 },
        { exchange: 'binance', symbol: 'BTCUSDT', quantity: 0 },
        { exchange: 'binance', symbol: 'BTCUSDT', quantity: 1 },
      ],
      'binance',
      'BTCUSDT',
    );
    expect(row?.quantity).toBe(1);
    expect(openPositionFor([{ exchange: 'nasdaq', symbol: 'AAPL', quantity: 1 }], 'binance', 'AAPL')).toBeUndefined();
    expect(isCashEquity('nasdaq')).toBe(true);
    expect(isCashEquity('binance')).toBe(false);
  });
});
