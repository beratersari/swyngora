import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { PumpsPage } from './PumpsPage';

const scan = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanPumpEventsQuery: (...args: unknown[]) => scan(...args),
  };
});

const hitPayload = {
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
};

describe('PumpsPage', () => {
  beforeEach(() => {
    scan.mockReset();
    refetch.mockReset();
  });

  it('maps nested event fields into table columns', async () => {
    const user = userEvent.setup();
    scan.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : hitPayload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<PumpsPage />, { routerEntries: ['/pumps'] });

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(await screen.findByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByText('12.50%')).toBeInTheDocument();
    expect(screen.getByText('4.25')).toBeInTheDocument();
    // locale-dependent time string — assert year present
    expect(screen.getByText(/2024/)).toBeInTheDocument();
  });

  it('second Scan with same filters calls refetch', async () => {
    const user = userEvent.setup();
    scan.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : hitPayload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<PumpsPage />, { routerEntries: ['/pumps'] });

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(await screen.findByText('ETH/USDT')).toBeInTheDocument();
    expect(refetch).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
