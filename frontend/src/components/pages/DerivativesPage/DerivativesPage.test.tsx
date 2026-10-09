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
            change24h: '1000000000',
            change24hDefinite: true,
            liqLong: '1000',
            liqShort: '2000',
            liqDefinite: false,
            longPct: '55',
            longDefinite: false,
            takerBuyPct: '62',
            takerDefinite: false,
            orderRatio: '+0.40',
            orderLean: 'buy',
            orderBuy: 12,
            orderSell: 4,
            orderDefinite: true,
            unusualMetric: 'funding',
            unusualBand: 'extreme_high',
            unusualDistance: '49',
            unusualVenue: 'binance',
            unusualDefinite: true,
            cvdSplit: 'price_up_cvd_down',
            cvdSplitDuration: '35m',
            cvdSplitSeconds: '2100',
            cvdSplitDefinite: true,
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
    expect(screen.getByRole('columnheader', { name: 'OI 24h' })).toBeInTheDocument();
    expect(screen.getByText('+1.00B')).toBeInTheDocument();
    expect(screen.getAllByTitle(/stored open-interest sample about 24 hours ago/i).length).toBeGreaterThan(0);
    expect(screen.getByRole('columnheader', { name: 'Recorded' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Crowd long' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Taker buy 1h' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Top − crowd' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'CVD day' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Price vs CVD' })).toBeInTheDocument();
    expect(screen.getByText('Price up, CVD down · 35m')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Print size 15m' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Order count 15m' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Unusual' })).toBeInTheDocument();
    expect(screen.getByText('+0.40 Buy')).toBeInTheDocument();
    expect(screen.getByText('Funding · Extremely high · Binance')).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Liq heat' })).toBeInTheDocument();
    expect(screen.getByRole('columnheader', { name: 'Bid 1%' })).toBeInTheDocument();
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

  it('colors one column and leaves an incomplete cell blank', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/derivatives" element={<DerivativesPage />} />
        <Route path="/markets/binance/BTCUSDT" element={<div>coin chart</div>} />
      </Routes>,
      { routerEntries: ['/derivatives'] },
    );

    await user.click(screen.getByText('Heatmap'));
    const blank = screen.getByTestId('derivatives-heat-cell');
    expect(blank).toHaveAttribute('data-tone', 'blank');
    expect(blank).toHaveTextContent('—');
    expect(screen.queryByRole('columnheader', { name: 'Recorded' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('combobox', { name: 'Color' }));
    await user.click(await screen.findByText('Price vs CVD'));
    const split = screen.getByTestId('derivatives-heat-cell');
    expect(split).toHaveAttribute('data-tone', 'up');
    expect(split).toHaveTextContent('+35m');

    await user.click(screen.getByRole('combobox', { name: 'Color' }));
    await user.click(await screen.findByText('Order count 15m'));
    const colored = screen.getByTestId('derivatives-heat-cell');
    expect(colored).toHaveAttribute('data-tone', 'up');
    expect(colored).toHaveTextContent('+0.40');

    await user.click(colored);
    expect(screen.getByText('coin chart')).toBeInTheDocument();
  });
});
