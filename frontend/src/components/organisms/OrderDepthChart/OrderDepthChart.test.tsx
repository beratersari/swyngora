import { beforeAll, describe, expect, it } from 'vitest';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { OrderDepthChart } from './OrderDepthChart';
import type { SpotOrderBook } from '@/libs/api';

const book: SpotOrderBook = {
  lastPrice: '100',
  bids: [{ price: '99', quantity: '1', cumulative: '1' }],
  asks: [{ price: '101', quantity: '2', cumulative: '2' }],
};

describe('OrderDepthChart', () => {
  it('renders the canvas and metric chips', async () => {
    const user = userEvent.setup();
    renderWithProviders(<OrderDepthChart book={book} />);
    expect(screen.getByTestId('order-depth-chart')).toBeInTheDocument();
    expect(screen.getByTestId('order-depth-canvas')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /notional|nominal/i }));
    expect(screen.getByRole('button', { name: /notional|nominal/i })).toBeInTheDocument();
  });

  it('shows empty copy when there is no book', () => {
    renderWithProviders(<OrderDepthChart book={{ bids: [], asks: [] }} />);
    expect(screen.getByText(/No depth|derinlik yok/i)).toBeInTheDocument();
  });
});
