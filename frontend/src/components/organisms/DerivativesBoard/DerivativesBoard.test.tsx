import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { renderWithProviders } from '@/test/render';
import type { DerivativesCoin } from '@/libs/api';
import { DerivativesBoard } from './DerivativesBoard';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        priceCvd: { kind: 'agrees' },
      },
      isLoading: false,
      isFetching: false,
      isError: false,
    }),
  };
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="loc">{location.pathname}</div>;
}

const standout: DerivativesCoin = {
  symbol: 'BTCUSDT',
  base: 'BTC',
  unusualMetric: 'funding',
  unusualBand: 'extreme_high',
  unusualVenue: 'binance',
  unusualDefinite: true,
};

const quiet: DerivativesCoin = {
  symbol: 'ETHUSDT',
  base: 'ETH',
  unusualDefinite: false,
};

describe('DerivativesBoard outlier situation', () => {
  it('opens the coin situation from an unusual standout and still opens the chart from the row', () => {
    renderWithProviders(
      <>
        <DerivativesBoard exchange="all" coins={[standout, quiet]} />
        <Routes>
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </>,
    );

    const open = screen.getByRole('button', { name: 'Funding · Extremely high · Binance' });
    expect(open).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('button', { name: 'ETH' })).not.toBeInTheDocument();

    fireEvent.click(open);
    expect(open).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('Combined situation for this coin.')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(screen.getByTestId('loc')).toHaveTextContent('/');

    fireEvent.click(screen.getByRole('button', { name: 'Funding · Extremely high · Binance' }));
    expect(screen.getByRole('button', { name: 'Funding · Extremely high · Binance' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );

    const quietRow = screen.getByRole('link', { name: 'ETH' }).closest('tr');
    expect(quietRow).not.toBeNull();
    fireEvent.click(quietRow as HTMLElement);
    expect(screen.getByTestId('loc')).toHaveTextContent('/markets/binance/ETHUSDT');
  });
});
