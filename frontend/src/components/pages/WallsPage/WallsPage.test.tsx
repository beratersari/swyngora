import { describe, expect, it, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { WallsPage } from './WallsPage';

const load = vi.fn();
const refetch = vi.fn();

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useScanSuspiciousWallsQuery: (...args: unknown[]) => load(...args),
  };
});

const payload = {
  summary: '1 suspicious walls and 1 brief flickers across 20 watched books',
  hits: [
    {
      exchange: 'binance',
      symbol: 'SOLUSDT',
      side: 'bid',
      price: '150',
      notional: '400000',
      distancePct: '0.4',
      kind: 'suspicious',
      appearCount: 5,
      present: true,
    },
    {
      exchange: 'bybit',
      symbol: 'ETHUSDT',
      side: 'ask',
      price: '2500',
      notional: '90000',
      distancePct: '0.2',
      kind: 'flicker',
      appearCount: 1,
      present: false,
    },
  ],
};

describe('WallsPage', () => {
  beforeEach(() => {
    load.mockReset();
    refetch.mockReset();
  });

  it('lists a suspicious wall ahead of a brief flicker', async () => {
    const user = userEvent.setup();
    load.mockImplementation((_args: unknown, opts?: { skip?: boolean }) => ({
      data: opts?.skip ? undefined : payload,
      isFetching: false,
      isSuccess: !opts?.skip,
      isError: false,
      refetch,
    }));
    renderWithProviders(<WallsPage />, { routerEntries: ['/walls'] });
    await user.click(screen.getByRole('button', { name: /load market|piyasayı yükle/i }));
    const symbols = await screen.findAllByText(/SOL\/USDT|ETH\/USDT/);
    expect(symbols[0]).toHaveTextContent('SOL/USDT');
    expect(symbols[1]).toHaveTextContent('ETH/USDT');
    expect(screen.getByText('400,000')).toBeInTheDocument();
    expect(screen.getByText('Flicker')).toBeInTheDocument();
    expect(screen.getByText('Gone')).toBeInTheDocument();
  });
});
