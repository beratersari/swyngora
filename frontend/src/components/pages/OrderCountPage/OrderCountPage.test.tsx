import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { OrderCountPage } from './OrderCountPage';

const load = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketOrderCountOverviewQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: '5m market is buy heavy',
  windows: [
    {
      window: '5m',
      ratio: '+0.40',
      buyCoins: 2,
      sellCoins: 1,
      balancedCoins: 0,
      quietCoins: 0,
      summary: '5m: more market buys',
    },
    { window: '1m', ratio: '+0.10', buyCoins: 1, sellCoins: 1, balancedCoins: 0, quietCoins: 0 },
    { window: '15m', ratio: '-0.20', buyCoins: 0, sellCoins: 2, balancedCoins: 0, quietCoins: 0 },
  ],
  coins: [
    {
      symbol: 'ETHUSDT',
      exchange: 'all',
      windows: [{ window: '5m', ratio: '-0.50', lean: 'sell', buyCount: 1, sellCount: 3, summary: 'ETH sells' }],
    },
    {
      symbol: 'BTCUSDT',
      exchange: 'all',
      windows: [{ window: '5m', ratio: '+0.60', lean: 'buy', buyCount: 8, sellCount: 2, summary: 'BTC buys' }],
    },
  ],
};

describe('OrderCountPage', () => {
  beforeEach(() => {
    load.mockReset();
    refetch.mockReset();
  });

  it('shows the market ratio and ranks coins buy-heavy first', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<OrderCountPage />, { routerEntries: ['/order-count'] });
    expect(load).toHaveBeenCalledWith(
      { exchange: 'all', quote: 'USDT', limit: 100 },
      expect.objectContaining({ skip: true }),
    );

    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    expect(await screen.findByText('+0.40')).toBeInTheDocument();
    const symbols = screen.getAllByText(/BTC\/USDT|ETH\/USDT/);
    expect(symbols[0]).toHaveTextContent('BTC/USDT');
    expect(screen.getByText('BTC buys')).toBeInTheDocument();
    expect(refetch).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
