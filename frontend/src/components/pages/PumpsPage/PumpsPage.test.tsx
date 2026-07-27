import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { PumpsPage } from './PumpsPage';

const scan = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanPumpEventsQuery: (...args: unknown[]) => scan(...args),
  };
});

describe('PumpsPage', () => {
  it('maps nested event fields into table columns', async () => {
    const user = userEvent.setup();
    scan.mockReturnValue({
      data: undefined,
      isFetching: false,
      isSuccess: false,
      isError: false,
    });
    renderWithProviders(<PumpsPage />, { routerEntries: ['/pumps'] });

    // Live API shape: bestReturnPct + events[], not flat returnPct on hit
    scan.mockReturnValue({
      data: {
        hitCount: 1,
        hits: [
          {
            symbol: 'ETHUSDT',
            exchange: 'binance',
            interval: '15m',
            bestReturnPct: 12.5,
            events: [
              {
                openTime: '2024-06-01T12:00:00Z',
                returnPct: 12.5,
                volumeRatio: 4.25,
              },
            ],
          },
        ],
      },
      isFetching: false,
      isSuccess: true,
      isError: false,
    });
    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(await screen.findByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByText('12.50%')).toBeInTheDocument();
    expect(screen.getByText('4.25')).toBeInTheDocument();
    // locale-dependent time string — assert year present
    expect(screen.getByText(/2024/)).toBeInTheDocument();
  });
});
