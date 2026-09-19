import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsTimeline } from './ForensicsTimeline';

describe('ForensicsTimeline', () => {
  it('renders beats in the given order', () => {
    renderWithProviders(
      <ForensicsTimeline
        timeline={{
          knownUntil: '2026-08-20T14:15:00Z',
          summary: 'Binance open interest started rising (about 8 minutes before).',
          events: [
            {
              at: '2026-08-20T13:50:00Z',
              offsetLabel: 'about 10 minutes before',
              kind: 'gamma',
              venue: 'deribit',
              title: 'Dealers were short gamma',
              detail: 'Spot was above the flip (97200).',
            },
            {
              at: '2026-08-20T13:52:00Z',
              offsetLabel: 'about 8 minutes before',
              kind: 'oi',
              venue: 'binance',
              title: 'Binance open interest started rising',
              detail: 'Kept going until price moved.',
            },
            {
              at: '2026-08-20T14:00:00Z',
              offsetLabel: 'at the event',
              kind: 'price',
              venue: 'binance',
              title: 'Binance price started moving up',
            },
            {
              at: '2026-08-20T14:02:00Z',
              offsetLabel: 'about 2 minutes after',
              kind: 'price',
              venue: 'bybit',
              title: 'Bybit price started moving up',
              orderUnknown: true,
            },
            {
              at: '2026-08-20T14:03:00Z',
              offsetLabel: 'about 3 minutes after',
              kind: 'liquidations',
              venue: 'binance',
              title: 'Binance short liquidations started',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText(/what happened, in order|zaman sırasıyla/i)).toBeInTheDocument();
    expect(screen.getByText(/known as of|itibarıyla bilinen/i)).toBeInTheDocument();
    expect(screen.getByText(/dealers were short gamma/i)).toBeInTheDocument();
    expect(screen.getByText(/above the flip/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Binance open interest started rising/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/Bybit price started moving up/i)).toBeInTheDocument();
    expect(screen.getByText(/order not certain|sıra kesin değil/i)).toBeInTheDocument();
    expect(screen.getByText(/short liquidations started/i)).toBeInTheDocument();
  });

  it('renders nothing without events', () => {
    const { container } = renderWithProviders(<ForensicsTimeline timeline={{ events: [] }} />);
    expect(container.querySelector('[aria-label]')).toBeNull();
  });
});
