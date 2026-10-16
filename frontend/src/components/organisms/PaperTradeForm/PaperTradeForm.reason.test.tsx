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
  });
});
