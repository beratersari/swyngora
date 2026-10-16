import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders, renderWithTheme } from '@/test/render';
import type { MarginPosition } from '@/libs/api';
import { PortfolioMarginPositionsTable } from './PortfolioMarginPositionsTable';
import type { MarginBracketUpdate } from './PortfolioMarginPositionsTable.types';

vi.mock('@/libs/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/libs/api')>();
  return {
    ...actual,
    useGetMarketSituationQuery: () => ({
      data: {
        symbol: 'BTCUSDT',
        priceCvd: { kind: 'agrees' },
      },
      isLoading: false,
      isFetching: false,
    }),
  };
});

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

  it('closes 4 of a 10 BTC position and leaves the rest', async () => {
    const onClose = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PortfolioMarginPositionsTable items={[{ ...long, quantity: 10 }]} onClose={onClose} />,
    );
    fireEvent.click(screen.getByRole('button', { name: /^close$|^kapat$/i }));
    const qty = await screen.findByRole('spinbutton', {
      name: /size to close|kapatılacak miktar/i,
    });
    fireEvent.change(qty, { target: { value: '4' } });
    fireEvent.blur(qty);
    expect(screen.getByText(/6 stays open|6 açık kalır/i)).toBeInTheDocument();
    const closeButtons = screen.getAllByRole('button', { name: /^close$|^kapat$/i });
    fireEvent.click(closeButtons[closeButtons.length - 1]);
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledWith('pos-1', 4);
    });
  });

  it('closes the whole position when the size is the open size', async () => {
    const onClose = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(<PortfolioMarginPositionsTable items={[long]} onClose={onClose} />);
    fireEvent.click(screen.getByRole('button', { name: /^close$|^kapat$/i }));
    const closeButtons = await screen.findAllByRole('button', { name: /^close$|^kapat$/i });
    fireEvent.click(closeButtons[closeButtons.length - 1]);
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledWith('pos-1', undefined);
    });
  });

  it('does not close more than the open size', async () => {
    const onClose = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PortfolioMarginPositionsTable items={[{ ...long, quantity: 10 }]} onClose={onClose} />,
    );
    fireEvent.click(screen.getByRole('button', { name: /^close$|^kapat$/i }));
    const qty = await screen.findByRole('spinbutton', {
      name: /size to close|kapatılacak miktar/i,
    });
    fireEvent.change(qty, { target: { value: '11' } });
    fireEvent.blur(qty);
    const closeButtons = screen.getAllByRole('button', { name: /^close$|^kapat$/i });
    fireEvent.click(closeButtons[closeButtons.length - 1]);
    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
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

describe('PortfolioMarginPositionsTable', () => {
  it('shows the placement read and the live read on an open margin position', () => {
    renderWithProviders(
      <PortfolioMarginPositionsTable
        items={[
          {
            id: 'm1',
            exchange: 'binance',
            symbol: 'BTCUSDT',
            side: 'long',
            leverage: 5,
            quantity: 1,
            status: 'open',
            entrySituation: {
              symbol: 'BTCUSDT',
              priceCvd: { kind: 'price_up_cvd_down' },
            },
          },
          {
            id: 'eq',
            exchange: 'nasdaq',
            symbol: 'AAPL',
            side: 'long',
            leverage: 2,
            quantity: 1,
            status: 'open',
            entrySituation: {
              symbol: 'AAPL',
              priceCvd: { kind: 'agrees' },
            },
          },
        ]}
      />,
    );
    expect(screen.getByText('Margin long 5x')).toBeInTheDocument();
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText('Margin long 2x')).not.toBeInTheDocument();
  });
});
