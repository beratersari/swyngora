import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ExecutionCompare as ExecutionCompareModel } from '@/libs/api';
import { renderWithTheme } from '@/test/render';
import { ExecutionCompare } from './ExecutionCompare';
import { formatPrice } from '@/libs/utils/formatPrice';
import { cheapestStyles, formatComparePct, legVenues } from './helpers';

const priced: ExecutionCompareModel = {
  side: 'buy',
  symbol: 'BTCUSDT',
  quantity: 2,
  exchange: 'binance',
  touchPrice: 100,
  touchExchange: 'binance',
  rows: [
    {
      style: 'market',
      status: 'filled',
      exchange: 'binance',
      slippagePct: 5,
      fee: 1,
      feePct: 0.1,
      averagePrice: 105,
      cashDebit: 211,
      legs: [{ exchange: 'binance', quantity: 2 }],
    },
    {
      style: 'best_cost',
      status: 'filled',
      exchange: 'all',
      slippagePct: 0,
      fee: 1,
      feePct: 0.35,
      averagePrice: 100,
      cashDebit: 201,
      savedVsMarket: 10,
      legs: [
        { exchange: 'binance', quantity: 1 },
        { exchange: 'coinbase', quantity: 1 },
      ],
    },
    {
      style: 'spread',
      status: 'filled',
      exchange: 'all',
      slippagePct: 0,
      fee: 1,
      feePct: 0.1,
      averagePrice: 100,
      cashDebit: 200,
      savedVsMarket: 11,
      assumption: 'refill',
      legs: [{ exchange: 'binance', quantity: 2 }],
    },
  ],
};

describe('ExecutionCompare', () => {
  it('shows fee and slippage for the three paths and marks the least cash', async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    renderWithTheme(
      <ExecutionCompare
        compare={priced}
        selected="market"
        minutes={10}
        slices={2}
        onSelect={onSelect}
        onMinutes={vi.fn()}
        onSlices={vi.fn()}
      />,
    );
    expect(screen.getByText(/same size, three ways|aynı boy, üç yol/i)).toBeInTheDocument();
    expect(screen.getByText('5%')).toBeInTheDocument();
    expect(screen.getByText(/least cash|en az nakit/i)).toBeInTheDocument();
    expect(screen.getByText(/if the book does not refill|defter dolmazsa/i)).toBeInTheDocument();
    expect(screen.getByText('binance 1, coinbase 1')).toBeInTheDocument();
    expect(screen.getAllByText('binance 2')).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: /over time|zamana yay/i }));
    expect(onSelect).toHaveBeenCalledWith('spread');
  });

  it('says when a path cannot fill and does not treat that as zero cash', () => {
    renderWithTheme(
      <ExecutionCompare
        compare={{
          side: 'buy',
          touchPrice: 100,
          touchExchange: 'binance',
          rows: [
            { style: 'market', status: 'filled', slippagePct: 0, fee: 1, feePct: 0.1, averagePrice: 100, cashDebit: 101 },
            { style: 'best_cost', status: 'slippage_limited', cashDebit: 0 },
            { style: 'spread', status: 'schedule', schedule: 'interval' },
          ],
        }}
        selected="market"
        minutes={1}
        slices={60}
        onSelect={vi.fn()}
        onMinutes={vi.fn()}
        onSlices={vi.fn()}
      />,
    );
    expect(screen.getByText(/slippage cap cannot hold|kayma sınırı bu boyu tutamaz/i)).toBeInTheDocument();
    expect(screen.getByText(/closer than 15 seconds|15 saniyeden yakın/i)).toBeInTheDocument();
    expect(screen.queryByText(/^0$/)).not.toBeInTheDocument();
  });
});

describe('cheapestStyles', () => {
  it('leaves a tie unmarked', () => {
    expect(
      cheapestStyles('buy', [
        { style: 'market', status: 'filled', cashDebit: 10 },
        { style: 'best_cost', status: 'filled', cashDebit: 10 },
      ]).size,
    ).toBe(0);
    expect(formatComparePct(0.5)).toBe('0.5%');
  });

  it('puts the size next to each exchange', () => {
    expect(
      legVenues({
        legs: [
          { exchange: 'binance', quantity: 0.4 },
          { exchange: 'bybit', quantity: 0.6 },
        ],
      }),
    ).toBe(`binance ${formatPrice(0.4)}, bybit ${formatPrice(0.6)}`);
  });
});
