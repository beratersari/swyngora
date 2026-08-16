import { beforeAll, describe, expect, it, vi } from 'vitest';

beforeAll(() => {
  Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
    value: () => null,
  });
});
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { OrderHeatmap } from './OrderHeatmap';
import type { OrderHeatmapData } from './OrderHeatmap.types';

const data: OrderHeatmapData = {
  symbol: 'BTCUSDT',
  groupSize: '1',
  windowSeconds: 600,
  sampleEveryMs: 2500,
  to: '2026-08-16T12:10:00.000Z',
  columns: [
    {
      t: '2026-08-16T12:09:00.000Z',
      mid: '100',
      bids: [{ price: '99', notional: '4000', isWall: true }],
      asks: [{ price: '101', notional: '1500' }],
    },
  ],
};

describe('OrderHeatmap', () => {
  it('renders the tape and window chips', async () => {
    const onWindowChange = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <OrderHeatmap data={data} windowSeconds={600} onWindowChange={onWindowChange} />,
    );
    expect(screen.getByTestId('order-heatmap')).toBeInTheDocument();
    expect(screen.getByTestId('order-heatmap-canvas')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '5m' }));
    expect(onWindowChange).toHaveBeenCalledWith(300);
  });

  it('shows empty copy when there is no tape yet', () => {
    renderWithProviders(
      <OrderHeatmap data={{ columns: [] }} windowSeconds={600} onWindowChange={() => undefined} />,
    );
    expect(screen.getByText(/No live book/i)).toBeInTheDocument();
  });
});
