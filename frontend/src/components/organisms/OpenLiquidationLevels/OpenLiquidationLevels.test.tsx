import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import type { LiquidationOpenLevels } from '@/libs/api';
import { formatPrice } from '@/libs/utils';
import { renderWithTheme } from '@/test/render';
import { OpenLiquidationLevels } from './OpenLiquidationLevels';

const report: LiquidationOpenLevels = {
  symbol: 'BTCUSDT',
  range: '24h',
  leverageMix: [{ leverage: 10, weight: 0.25 }],
  missingVenues: ['okx'],
  combined: {
    exchange: 'combined',
    definite: false,
    segments: [],
    cumulative: [],
    openLong: 0,
    openShort: 0,
    delta: 0,
    hitLots: 0,
    closedLots: 0,
  },
  binance: {
    exchange: 'binance',
    definite: true,
    openLong: 500_000,
    openShort: 250_000,
    delta: 250_000,
    hitLots: 3,
    closedLots: 1,
    segments: [
      {
        exchange: 'binance',
        side: 'long',
        leverage: 25,
        price: 90_000,
        notional: 400_000,
        openedAt: '2026-09-29T12:00:00Z',
        status: 'open',
      },
      {
        exchange: 'binance',
        side: 'short',
        leverage: 10,
        price: 110_000,
        notional: 100_000,
        openedAt: '2026-09-29T12:00:00Z',
        closedAt: '2026-09-29T13:00:00Z',
        status: 'hit',
      },
    ],
  },
};

describe('OpenLiquidationLevels', () => {
  it('lists open bands and the still-open size', () => {
    renderWithTheme(
      <OpenLiquidationLevels data={report} venue="binance" range="24h" />,
    );
    expect(screen.getByTestId('open-liquidation-levels')).toBeInTheDocument();
    expect(screen.getByTestId('open-level-mix')).toHaveTextContent(/10× 25%/);
    expect(screen.getByTestId('open-level-long')).toHaveTextContent(/500\.00K/);
    expect(screen.getByTestId('open-level-short')).toHaveTextContent(/250\.00K/);
    expect(screen.getByTestId('open-level-delta')).toHaveTextContent(/250\.00K/);
    expect(screen.getByTestId('open-level-hits')).toHaveTextContent(/3/);
    expect(screen.getByTestId('open-level-closed')).toHaveTextContent(/1/);
    const rows = screen.getAllByTestId('open-level-row');
    expect(rows).toHaveLength(1);
    expect(rows[0]).toHaveTextContent(/Long/);
    expect(rows[0]).toHaveTextContent(/25×/);
    expect(rows[0]).toHaveTextContent(formatPrice(90_000));
    expect(screen.queryByTestId('open-level-withheld')).not.toBeInTheDocument();
  });

  it('withholds the combined book when a venue is missing', () => {
    renderWithTheme(
      <OpenLiquidationLevels data={report} venue="combined" range="24h" />,
    );
    expect(screen.getByTestId('open-level-withheld')).toBeInTheDocument();
    expect(screen.getByTestId('open-level-missing')).toHaveTextContent(/OKX/);
    expect(screen.queryByTestId('open-level-row')).not.toBeInTheDocument();
    expect(screen.queryByTestId('open-level-long')).not.toBeInTheDocument();
  });
});
