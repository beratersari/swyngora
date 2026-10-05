import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { PrintSizePage } from './PrintSizePage';

const load = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketPrintSizeOverviewQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: '15m prints got larger',
  windows: [
    {
      window: '15m',
      buy: { avg: '105000', priorAvg: '30000' },
      sell: { avg: '10000', priorAvg: '10000' },
      read: 'larger',
      largerCoins: 1,
      smallerCoins: 1,
      fewLargeCoins: 0,
      manySmallCoins: 0,
      summary: 'Last 15m: buy prints averaged 105000 USDT vs 30000 before.',
    },
    {
      window: '5m',
      buy: { avg: '90000', priorAvg: '40000' },
      sell: { avg: '10000', priorAvg: '10000' },
      largerCoins: 1,
      smallerCoins: 0,
      fewLargeCoins: 0,
      manySmallCoins: 0,
    },
  ],
  coins: [
    {
      symbol: 'ETHUSDT',
      exchange: 'all',
      windows: [
        {
          window: '15m',
          buy: { avg: '10000', priorAvg: '40000', ratio: '0.25' },
          sell: { avg: '10000', priorAvg: '10000', ratio: '1' },
          read: 'smaller',
        },
      ],
    },
    {
      symbol: 'BTCUSDT',
      exchange: 'all',
      windows: [
        {
          window: '15m',
          buy: { avg: '200000', priorAvg: '20000', ratio: '10' },
          sell: { avg: '10000', priorAvg: '10000', ratio: '1' },
          read: 'few_large',
        },
      ],
    },
  ],
};

describe('PrintSizePage', () => {
  beforeEach(() => {
    load.mockReset();
    refetch.mockReset();
  });

  it('shows the market average and ranks the larger coin first', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<PrintSizePage />, { routerEntries: ['/print-size'] });
    expect(load).toHaveBeenCalledWith(
      { exchange: 'all', quote: 'USDT', limit: 100 },
      expect.objectContaining({ skip: true }),
    );

    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    expect((await screen.findAllByText(/105000/)).length).toBeGreaterThan(0);
    const symbols = screen.getAllByText(/BTC\/USDT|ETH\/USDT/);
    expect(symbols[0]).toHaveTextContent('BTC/USDT');
    expect(screen.getByText('200000')).toBeInTheDocument();
    expect(refetch).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
