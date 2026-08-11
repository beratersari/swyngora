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
    expect(screen.getByTestId('ob-bid-100')).toHaveTextContent('|');
    expect(screen.getByTestId('ob-bid-100')).toHaveTextContent('100.0');
    expect(screen.getAllByText(/wall/i).length).toBeGreaterThan(0);
    expect(screen.getByRole('combobox', { name: 'Price group' })).toBeInTheDocument();
    expect(onGroupChange).not.toHaveBeenCalled();
  });

  it('pads tiny group steps so decimals share one column', () => {
    renderWithProviders(
      <OrderBookPanel
        book={{
          ...book,
          lastPrice: '0.00001234',
          groupSize: '0.00001',
          suggestedGroupSizes: ['0.00000001', '0.0000001', '0.000001', '0.00001', '0.0001'],
        }}
        group="0.00001"
        onGroupChange={vi.fn()}
      />,
    );
    expect(screen.getByRole('option', { name: '0.00000001' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.00000010' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.00000100' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.00001000' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.00010000' })).toBeInTheDocument();
  });

  it('does not keep a leftover finer group than the server suggests', () => {
    renderWithProviders(
      <OrderBookPanel
        book={{
          ...book,
          lastPrice: '0.0237',
          groupSize: '0.000005',
          suggestedGroupSizes: ['0.000001', '0.000005', '0.00001', '0.0001'],
        }}
        group="0.00000001"
        onGroupChange={vi.fn()}
      />,
    );
    expect(screen.queryByRole('option', { name: '0.00000001' })).not.toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.000001' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: '0.000100' })).toBeInTheDocument();
  });
});
