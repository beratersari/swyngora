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
      pairs: [
        { left: 'ADA', right: 'BTC', corr: 0.1, relation: 'mixed', samples: 30, complete: true },
        { left: 'BTC', right: 'ETH', corr: -0.95, relation: 'inverse', samples: 30, complete: true },
        { left: 'ETH', right: 'XRP', corr: 0.4, relation: 'loose', samples: 30, complete: true },
      ],
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
    const text = screen.getByTestId('portfolio-correlation').textContent ?? '';
    const strongest = text.indexOf('BTC and ETH: -0.95');
    const middle = text.indexOf('ETH and XRP: 0.40');
    const weakest = text.indexOf('ADA and BTC: 0.10');
    expect(strongest).toBeGreaterThanOrEqual(0);
    expect(strongest).toBeLessThan(middle);
    expect(middle).toBeLessThan(weakest);
    expect(screen.getByText(/SOL/)).toBeInTheDocument();
  });
});
