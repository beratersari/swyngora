import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { SurgesPage } from './SurgesPage';

const scan = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanMarketVolumeSurgesQuery: (...args: unknown[]) => scan(...args),
  };
});

const hitPayload = {
  summary: 'ETH is 4.2× typical on 15m',
  hits: [
    {
      symbol: 'ETHUSDT',
      exchange: 'binance',
      maxRatio: '4.2',
      hottest: '15m',
      grade: 'high',
      dominant: 'buy',
      summary: '15m is 4.2× typical',
    },
  ],
};

describe('SurgesPage', () => {
  beforeEach(() => {
    scan.mockReset();
    refetch.mockReset();
  });

  it('maps scan hits into table columns', async () => {
    const user = userEvent.setup();
    scan.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : hitPayload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<SurgesPage />, { routerEntries: ['/surges'] });

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(await screen.findByText('ETH/USDT')).toBeInTheDocument();
    expect(screen.getByText('4.2×')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
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
    renderWithProviders(<SurgesPage />, { routerEntries: ['/surges'] });

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(await screen.findByText('ETH/USDT')).toBeInTheDocument();
    expect(refetch).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: /scan|tara/i }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
