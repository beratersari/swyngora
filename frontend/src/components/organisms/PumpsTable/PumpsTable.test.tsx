import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { PumpsTable } from './PumpsTable';

describe('PumpsTable', () => {
  it('renders rows and opens on click', async () => {
    const onRowOpen = vi.fn();
    renderWithProviders(
      <PumpsTable
        hasScanned
        rows={[
          {
            exchange: 'binance',
            symbol: 'BTCUSDT',
            interval: '15m',
            returnPct: 12.5,
            volumeRatio: 2.1,
            openTime: '2024-01-01T00:00:00Z',
            eventCount: 1,
          },
        ]}
        emptyHint="hint"
        emptyTitle="empty"
        columns={{
          symbol: 'Symbol',
          returnPct: 'Return',
          volumeRatio: 'Vol',
          time: 'Time',
          events: 'Events',
        }}
        onRowOpen={onRowOpen}
        locale="en-US"
      />,
    );
    expect(screen.getByText(/BTC/)).toBeInTheDocument();
    await userEvent.click(screen.getByText(/BTC/));
    expect(onRowOpen).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });
});
