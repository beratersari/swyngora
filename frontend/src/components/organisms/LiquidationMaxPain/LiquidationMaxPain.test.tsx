import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { LiquidationMaxPain } from './LiquidationMaxPain';
import type { MaxPainReport } from './LiquidationMaxPain.types';

vi.mock('@/libs/hooks', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/hooks')>();
  return {
    ...actual,
    useDisplayCurrency: () => ({
      formatCompact: (v: string | number | null | undefined) => String(v ?? '—'),
      formatPrice: (v: string | number | null | undefined) => String(v ?? '—'),
    }),
  };
});

const sample: MaxPainReport = {
  symbol: 'BTCUSDT',
  summary: 'Largest short pocket is 8.00M above (+2.50%). Largest long pocket is 5.00M below (−1.80%).',
  venues: [
    {
      exchange: 'binance',
      price: '64000',
      openInterestValue: '100000000',
      above: {
        side: 'short',
        direction: 'up',
        price: '65600',
        movePct: '+2.50%',
        notional: '8000000',
        leverage: '25',
      },
      below: {
        side: 'long',
        direction: 'down',
        price: '62848',
        movePct: '-1.80%',
        notional: '5000000',
        leverage: '50',
      },
      aboveLevels: [
        { price: '65600', movePct: '+2.50%', notional: '8000000' },
        { price: '67200', movePct: '+5.00%', notional: '1200000' },
      ],
    },
  ],
};

describe('LiquidationMaxPain', () => {
  it('shows the largest pocket above and below last price', async () => {
    renderWithTheme(<LiquidationMaxPain data={sample} />);
    expect(await screen.findByTestId('liquidation-max-pain')).toBeInTheDocument();
    expect(screen.getByText(/largest short pocket/i)).toBeInTheDocument();
    expect(screen.getByTestId('liquidation-max-pain-up')).toHaveTextContent('65600');
    expect(screen.getByTestId('liquidation-max-pain-up')).toHaveTextContent('8000000');
    expect(screen.getByTestId('liquidation-max-pain-down')).toHaveTextContent('62848');
    expect(screen.getByTestId('liquidation-max-pain-down')).toHaveTextContent('5000000');
    expect(screen.getByTestId('liquidation-max-pain-up')).toHaveTextContent('67200');
  });
});
