import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import tr from '@/libs/i18n/locales/tr/unusual.json';
import { UnusualPage } from './UnusualPage';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanMarketUnusualQuery: () => ({
      data: {
        exchange: 'all',
        metric: 'all',
        universe: 36,
        scanned: 36,
        skipped: 0,
        quiet: 34,
        hits: [
          {
            symbol: 'BTCUSDT',
            base: 'BTC',
            exchange: 'binance',
            metric: 'funding',
            band: 'extreme_high',
            percentile: 99,
            distance: 49,
            current: 0.0003,
            samples: 24,
            spanHours: 192,
            unit: 'rate',
            label: 'settled',
          },
          {
            symbol: 'ETHUSDT',
            base: 'ETH',
            exchange: 'okx',
            metric: 'open_interest',
            band: 'low',
            percentile: 8,
            distance: 42,
            current: -3.2,
            samples: 20,
            spanHours: 480,
            unit: 'pct_change',
          },
        ],
      },
      isSuccess: true,
      isLoading: false,
      isError: false,
    }),
  };
});

describe('UnusualPage', () => {
  it('lists standout coins and opens the tape', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/unusual" element={<UnusualPage />} />
        <Route path="/markets/binance/BTCUSDT" element={<div>btc tape</div>} />
      </Routes>,
      { routerEntries: ['/unusual'] },
    );

    expect(screen.getByRole('columnheader', { name: 'Versus its past' })).toBeInTheDocument();
    expect(screen.getByText('Extremely high')).toBeInTheDocument();
    expect(screen.getByText('0.0300%')).toBeInTheDocument();
    expect(screen.getByText('36 coins scored. 2 stand out. 34 are typical or still short of samples.')).toBeInTheDocument();
    expect(screen.queryByText(/print size/i)).not.toBeInTheDocument();

    const link = screen.getByRole('link', { name: 'BTC' });
    expect(link).toHaveAttribute('href', '/markets/binance/BTCUSDT?tab=tape');
    await user.click(link);
    expect(screen.getByText('btc tape')).toBeInTheDocument();
  });

  it('ships Turkish copy for the page', () => {
    expect(tr.title).toBe('Olağandışı');
    expect(tr.bands.extreme_high).toBe('Aşırı yüksek');
    expect(tr.empty).toContain('olağan');
  });
});
