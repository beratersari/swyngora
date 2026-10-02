import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import { DerivativesPage } from './DerivativesPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketDerivativesBoardQuery: () => ({
      data: {
        definite: false,
        summary: 'No market read. Missing: okx.',
        coverage: {
          complete: false,
          missing: ['okx'],
          stale: [],
          venues: [
            { exchange: 'binance', status: 'fresh' },
            { exchange: 'bybit', status: 'fresh' },
            { exchange: 'okx', status: 'missing' },
          ],
        },
        coins: [
          {
            symbol: 'BTCUSDT',
            base: 'BTC',
            price: '100000',
            priceExchange: 'binance',
            openInterest: '8000000000',
            openInterestDefinite: false,
            fundingRatePct: '+0.01',
            fundingDefinite: false,
            change1h: '+100',
            change1hDefinite: false,
            change24hDefinite: false,
            liqLong: '1000',
            liqShort: '2000',
            liqDefinite: false,
            longPct: '55',
            longDefinite: false,
            takerBuyPct: '62',
            takerDefinite: false,
            venues: [
              { exchange: 'binance', oiStatus: 'fresh', oiValue: '8000000000' },
              { exchange: 'okx', oiStatus: 'missing' },
            ],
          },
        ],
      },
      isSuccess: true,
      isLoading: false,
      isError: false,
    }),
  };
});

describe('DerivativesPage', () => {
  it('shows the coin and withholds a combined total when a venue is missing', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/derivatives" element={<DerivativesPage />} />
        <Route path="/markets/binance/BTCUSDT" element={<div>coin chart</div>} />
      </Routes>,
      { routerEntries: ['/derivatives'] },
    );
    const link = screen.getByRole('link', { name: 'BTC' });
    expect(link).toHaveAttribute('href', '/markets/binance/BTCUSDT');
    expect(screen.getByRole('columnheader', { name: 'Crowd long' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Taker buy 1h' })).toBeInTheDocument();
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    expect(screen.queryByText('8.00B')).not.toBeInTheDocument();
    expect(screen.queryByText('55%')).not.toBeInTheDocument();
    expect(screen.queryByText('62%')).not.toBeInTheDocument();
    expect(screen.getByTestId('venue-coverage')).toBeInTheDocument();
    const row = link.closest('tr');
    expect(row).not.toBeNull();
    await user.click(row as HTMLElement);
    expect(screen.getByText('coin chart')).toBeInTheDocument();
  });
});
