import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import type { OrderPreview } from '@/libs/api';
import { PaperTradeForm } from './PaperTradeForm';

const preview = {
  product: 'spot',
  type: 'market',
  side: 'buy',
  exchange: 'binance',
  symbol: 'BTCUSDT',
  quantity: 0.01,
  situationStatus: 'ready',
  situation: {
    priceCvd: { kind: 'price_up_cvd_down' },
    openInterest: { flow: 'opened' },
  },
} as OrderPreview;

describe('PaperTradeForm reason', () => {
  it('sends only the clauses that stay checked', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onDraftChange = vi.fn();
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        preview={preview}
        onDraftChange={onDraftChange}
        onSubmit={onSubmit}
      />,
    );

    expect(screen.getByText('Reason for this order')).toBeInTheDocument();
    const boxes = screen.getAllByRole('checkbox');
    expect(boxes).toHaveLength(2);
    expect(boxes[0]).toBeChecked();
    expect(boxes[1]).toBeChecked();
    fireEvent.click(boxes[1]);

    const spin = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(spin, { target: { value: '0.01' } });
    fireEvent.blur(spin);
    fireEvent.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    const arg = onSubmit.mock.calls[0][0] as { reason?: string[] };
    expect(arg.reason).toEqual(['priceCvd']);
  });

  it('says a failed read will not be stored', () => {
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        preview={{ ...preview, situationStatus: 'failed', situation: undefined }}
        onSubmit={vi.fn()}
      />,
    );
    expect(
      screen.getByText('The market read did not load. This order will not store a reason.'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Past closes with this reason')).not.toBeInTheDocument();
  });

  it('shows past closes for the clauses that stay checked', () => {
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        preview={{
          ...preview,
          reasonHistory: [
            {
              clause: 'priceCvd',
              kind: 'price_up_cvd_down',
              still: { count: 1, profitable: 1, realizedPnL: 20, closes: [] },
              left: { count: 1, profitable: 0, realizedPnL: -10, closes: [] },
            },
            {
              clause: 'openInterest',
              kind: 'opened',
              still: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
              left: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
            },
          ],
        }}
        onSubmit={vi.fn()}
      />,
    );
    expect(screen.getByText('Past closes with this reason')).toBeInTheDocument();
    expect(screen.getByText(/Still the same: 1 close/)).toBeInTheDocument();
    expect(screen.getByText('No past close stored this reason.')).toBeInTheDocument();

    const boxes = screen.getAllByRole('checkbox');
    fireEvent.click(boxes[0]);
    expect(screen.queryByText(/Still the same: 1 close/)).not.toBeInTheDocument();
    expect(screen.getByText('No past close stored this reason.')).toBeInTheDocument();

    fireEvent.click(boxes[1]);
    expect(screen.queryByText('Past closes with this reason')).not.toBeInTheDocument();
  });

  it('hides past closes on a sell', () => {
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        defaultSide="sell"
        preview={{
          ...preview,
          side: 'sell',
          reasonHistory: [
            {
              clause: 'priceCvd',
              kind: 'price_up_cvd_down',
              still: { count: 1, profitable: 1, realizedPnL: 20, closes: [] },
              left: { count: 0, profitable: 0, realizedPnL: 0, closes: [] },
            },
          ],
        }}
        onSubmit={vi.fn()}
      />,
    );
    expect(screen.queryByText('Past closes with this reason')).not.toBeInTheDocument();
  });
});
