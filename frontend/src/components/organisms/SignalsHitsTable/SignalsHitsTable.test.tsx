import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SignalsHitsTable } from './SignalsHitsTable';

describe('SignalsHitsTable', () => {
  it('renders a hit row', () => {
    renderWithProviders(
      <SignalsHitsTable
        items={[
          {
            id: '1',
            ruleId: 'r',
            exchange: 'binance',
            symbol: 'ETHUSDT',
            ruleType: 'rsi',
            interval: '4h',
            marketDataKey: '2026-08-01T00:00:00Z',
            matchedAt: '2026-08-01T00:01:00Z',
            summary: 'RSI(14)=32.00 below 40',
          },
        ]}
      />,
    );
    expect(screen.getByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByText(/RSI\(14\)=32/)).toBeInTheDocument();
    expect(screen.getByText(/2026/)).toBeInTheDocument();
    expect(screen.getByText(/00:00:00/)).toBeInTheDocument();
  });
});
