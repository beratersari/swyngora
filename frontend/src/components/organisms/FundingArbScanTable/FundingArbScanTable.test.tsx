import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { FundingArbScanTable } from './FundingArbScanTable';

describe('FundingArbScanTable', () => {
  it('renders a scan hit', () => {
    renderWithProviders(
      <FundingArbScanTable
        scan={{
          hits: [
            {
              symbol: 'BTCUSDT',
              longExchange: 'binance',
              shortExchange: 'bybit',
              spreadPct: '0.02',
              netHorizonAfterRoundTrip: '12.5',
              summary: 'Long Binance, short Bybit',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('BTCUSDT')).toBeInTheDocument();
    expect(screen.getByText(/long binance/i)).toBeInTheDocument();
  });
});
