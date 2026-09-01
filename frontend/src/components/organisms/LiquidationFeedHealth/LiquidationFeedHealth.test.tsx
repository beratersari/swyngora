import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { LiquidationFeedHealth } from './LiquidationFeedHealth';

describe('LiquidationFeedHealth', () => {
  it('shows last print and a missing venue', () => {
    renderWithProviders(
      <LiquidationFeedHealth
        feed={{
          missing: ['bybit'],
          venues: [
            {
              exchange: 'binance',
              live: true,
              lastEventAt: '2026-08-30T12:00:05.000Z',
              lastSeenAt: '2026-08-30T12:00:08.000Z',
              coverageSeconds: 3600,
              gaps: [],
            },
            {
              exchange: 'bybit',
              live: false,
              lastEventAt: '2026-08-30T10:00:00.000Z',
              coverageSeconds: 0,
              gaps: [{ from: '2026-08-30T10:05:00.000Z', seconds: 7200 }],
            },
          ],
        }}
      />,
    );
    expect(screen.getByTestId('liquidation-feed-health')).toBeInTheDocument();
    expect(screen.getByText(/12:00:05Z/)).toBeInTheDocument();
    expect(screen.getByText(/no data from bybit/i)).toBeInTheDocument();
  });
});
