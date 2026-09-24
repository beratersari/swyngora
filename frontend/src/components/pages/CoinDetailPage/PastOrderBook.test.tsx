import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { PastOrderBook } from './PastOrderBook';

vi.mock('@/libs/api', async () => {
  const actual = await vi.importActual<typeof import('@/libs/api')>('@/libs/api');
  return {
    ...actual,
    useGetOrderBookHistoryQuery: (arg: { at?: string }) =>
      arg.at
        ? {
            data: {
              snapshot: {
                flashes: [
                  { side: 'bid', price: '100', notional: '8000', durationSec: 10, seenFrom: '2026-09-22T12:00:05Z' },
                ],
                moves: [
                  {
                    side: 'ask',
                    price: '101',
                    change: 'gained',
                    fromNotional: '200',
                    toNotional: '200',
                    peakNotional: '900',
                    seenFrom: '2026-09-22T12:00:08Z',
                  },
                ],
              },
            },
            isFetching: false,
          }
        : {
            data: { snapshots: [{ sampledAt: '2026-09-22T12:00:00Z', mid: '100' }] },
            isFetching: false,
          },
  };
});

describe('PastOrderBook', () => {
  it('shows a wall that vanished between minute snapshots', () => {
    renderWithProviders(<PastOrderBook symbol="BTCUSDT" exchange="binance" />);
    expect(screen.getByTestId('past-order-book')).toBeInTheDocument();
    expect(screen.getByText(/10s|10 sn/)).toBeInTheDocument();
    expect(screen.getByText(/900/)).toBeInTheDocument();
  });
});
