import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PriceDiffOpportunityTable } from './PriceDiffOpportunityTable';

describe('PriceDiffOpportunityTable', () => {
  it('shows duration, peak, and close reason', () => {
    renderWithProviders(
      <PriceDiffOpportunityTable
        status="all"
        onStatusChange={vi.fn()}
        onExpand={vi.fn()}
        opportunities={[
          {
            id: 'o1',
            symbol: 'ETHUSDT',
            buyExchange: 'binance',
            sellExchange: 'bybit',
            status: 'closed',
            closeReason: 'below_min_profit',
            review: {
              openDurationSec: 90,
              openNetDiffPct: 0.4,
              peakNetDiffPct: 0.81,
              lastNetDiffPct: 0.08,
              summary: 'Open 1m 30s. Peaked at 0.810%.',
              closeReason: 'below_min_profit',
            },
          },
        ]}
      />,
    );
    expect(screen.getByText(/ETHUSDT/i)).toBeInTheDocument();
    expect(screen.getByText(/1m 30s/)).toBeInTheDocument();
    expect(screen.getByText(/0.810%/)).toBeInTheDocument();
    expect(screen.getByText(/profit floor|kâr tabanı/i)).toBeInTheDocument();
  });
});
