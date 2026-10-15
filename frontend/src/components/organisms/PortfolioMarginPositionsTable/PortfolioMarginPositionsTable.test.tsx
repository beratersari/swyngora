import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import type { MarginPosition } from '@/libs/api';
import { PortfolioMarginPositionsTable } from './PortfolioMarginPositionsTable';
import type { MarginBracketUpdate } from './PortfolioMarginPositionsTable.types';

const long: MarginPosition = {
  id: 'pos-1',
  exchange: 'binance',
  symbol: 'BTCUSDT',
  side: 'long',
  status: 'open',
  quantity: 0.1,
  entryPrice: 100,
  leverage: 2,
  markPrice: 105,
  liquidationPrice: 50,
  margin: 5,
  stopLoss: 90,
  takeProfit: 120,
};

function renderTable(onBrackets = vi.fn().mockResolvedValue(undefined)) {
  renderWithTheme(
    <PortfolioMarginPositionsTable items={[long]} onBrackets={onBrackets} onClose={vi.fn()} />,
  );
  return onBrackets;
}

describe('PortfolioMarginPositionsTable brackets', () => {
  it('shows the stop and take profit on the open position', () => {
    renderTable();
    expect(screen.getByRole('cell', { name: '90' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: '120' })).toBeInTheDocument();
  });

  it('sends a changed stop and leaves the take profit', async () => {
    const onBrackets = renderTable();
    fireEvent.click(screen.getByRole('button', { name: /edit stop|zarar kesi düzenle/i }));
    const stop = await screen.findByRole('spinbutton', { name: /stop loss|zarar kes/i });
    fireEvent.change(stop, { target: { value: '80' } });
    fireEvent.blur(stop);
    fireEvent.click(screen.getByRole('button', { name: /^save$|^kaydet$/i }));
    await waitFor(() => {
      expect(onBrackets).toHaveBeenCalled();
    });
    const arg = onBrackets.mock.calls[0][0] as MarginBracketUpdate;
    expect(arg).toEqual({ id: 'pos-1', stopLoss: 80 });
  });

  it('removes the stop and keeps the take profit', async () => {
    const onBrackets = renderTable();
    fireEvent.click(screen.getByRole('button', { name: /edit stop|zarar kesi düzenle/i }));
    fireEvent.click(await screen.findByRole('button', { name: /remove stop|zarar kesi kaldır/i }));
    fireEvent.click(screen.getByRole('button', { name: /^save$|^kaydet$/i }));
    await waitFor(() => {
      expect(onBrackets).toHaveBeenCalledWith({ id: 'pos-1', clearStopLoss: true });
    });
  });

  it('does not save a long stop at or above the entry', async () => {
    const onBrackets = renderTable();
    fireEvent.click(screen.getByRole('button', { name: /edit stop|zarar kesi düzenle/i }));
    const stop = await screen.findByRole('spinbutton', { name: /stop loss|zarar kes/i });
    fireEvent.change(stop, { target: { value: '110' } });
    fireEvent.blur(stop);
    fireEvent.click(screen.getByRole('button', { name: /^save$|^kaydet$/i }));
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(onBrackets).not.toHaveBeenCalled();
  });

  it('hides edit and close on a closed position', () => {
    renderWithTheme(
      <PortfolioMarginPositionsTable
        items={[{ ...long, status: 'closed' }]}
        onBrackets={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.queryByRole('button', { name: /edit stop|zarar kesi düzenle/i })).toBeNull();
    expect(screen.queryByRole('button', { name: /^close$|^kapat$/i })).toBeNull();
  });
});
