import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OrderBookPanel } from './OrderBookPanel';
import type { SpotOrderBook } from '@/libs/api';

const book: SpotOrderBook = {
  lastPrice: '100.05',
  spread: '0.1',
  spreadPct: '0.1',
  groupSize: '0.1',
  suggestedGroupSizes: ['0.01', '0.1', '1'],
  bids: [
    { price: '100', quantity: '2', notional: '200', cumulative: '2', isWall: false },
    { price: '99.5', quantity: '40', notional: '3980', cumulative: '42', isWall: true },
  ],
  asks: [
    { price: '100.1', quantity: '1', notional: '100.1', cumulative: '1', isWall: false },
    { price: '100.5', quantity: '3', notional: '301.5', cumulative: '4', isWall: false },
  ],
  imbalance: 0.4,
};

describe('OrderBookPanel', () => {
  it('renders grouped bids, asks, walls, and group steps', async () => {
    const onGroupChange = vi.fn();
    renderWithProviders(
      <OrderBookPanel book={book} group="0.1" onGroupChange={onGroupChange} />,
    );
    expect(screen.getByTestId('order-book')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
    expect(screen.getAllByText(/wall/i).length).toBeGreaterThan(0);
    expect(screen.getByLabelText('Price group')).toBeInTheDocument();
    expect(onGroupChange).not.toHaveBeenCalled();
  });
});
