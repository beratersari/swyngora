import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PortfolioCorrelation } from './PortfolioCorrelation';

vi.mock('@/libs/api', () => ({
  useGetPortfolioCorrelationQuery: () => ({
    isLoading: false,
    isError: false,
    data: {
      window: '24h',
      interval: '5m',
      gross: 200,
      samples: 30,
      together: 0.91,
      swingPct: 4.2,
      apartPct: 2.1,
      holdings: [
        { asset: 'BTC', notional: 100, side: 'long' },
        { asset: 'ETH', notional: 100, side: 'long' },
      ],
      pairs: [{ left: 'BTC', right: 'ETH', corr: 0.91, relation: 'follows', samples: 30, complete: true }],
      missing: ['SOL'],
      note: 'note',
    },
  }),
}));

describe('PortfolioCorrelation', () => {
  it('says coins that move together do not spread the risk', () => {
    renderWithTheme(<PortfolioCorrelation portfolioId="b1" />);
    expect(screen.getByText(/moving together \(0\.91\)/i)).toBeInTheDocument();
    expect(screen.getByText(/4\.2%/)).toBeInTheDocument();
    expect(screen.getByText(/BTC and ETH: 0\.91/i)).toBeInTheDocument();
    expect(screen.getByText(/SOL/)).toBeInTheDocument();
  });
});
