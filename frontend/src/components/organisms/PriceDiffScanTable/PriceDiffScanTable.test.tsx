import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PriceDiffScanTable } from './PriceDiffScanTable';

describe('PriceDiffScanTable', () => {
  it('renders ranked routes', () => {
    renderWithProviders(
      <PriceDiffScanTable
        scan={{
          routes: [
            {
              buyExchange: 'binance',
              sellExchange: 'bybit',
              profitAfterFees: '18.2',
              profitPct: 0.18,
              usedNotional: '9000',
              executable: true,
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('binance → bybit')).toBeInTheDocument();
    expect(screen.getByText('0.180%')).toBeInTheDocument();
  });
});
