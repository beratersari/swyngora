import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import type { MarketUnusualScanHit } from '@/libs/api';
import { UnusualScanBoard } from './UnusualScanBoard';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        funding: { payer: 'long', ratePct: 0.01 },
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
  };
});

const hit: MarketUnusualScanHit = {
  symbol: 'BTCUSDT',
  base: 'BTC',
  exchange: 'okx',
  metric: 'funding',
  band: 'extreme_high',
  percentile: 99,
  distance: 49,
  samples: 24,
  spanHours: 48,
};

describe('UnusualScanBoard', () => {
  it('opens the coin situation from the standout band and keeps the tape link', () => {
    renderWithProviders(<UnusualScanBoard hits={[hit]} />);

    const tape = screen.getByRole('link', { name: 'BTC' });
    expect(tape).toHaveAttribute('href', '/markets/binance/BTCUSDT?tab=tape');

    const open = screen.getByRole('button', { name: 'Extremely high' });
    fireEvent.click(open);
    expect(open).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('Combined situation for this coin.')).toBeInTheDocument();
    expect(screen.getByText('Longs pay the combined funding (0.01%).')).toBeInTheDocument();
    expect(tape).toHaveAttribute('href', '/markets/binance/BTCUSDT?tab=tape');
  });
});
