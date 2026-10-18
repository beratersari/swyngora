import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PaperTradeForm } from './PaperTradeForm';

describe('PaperTradeForm take-profit targets', () => {
  it('puts a 25/25/50 scale-out on the OCO draft and the submit', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onDraftChange = vi.fn();
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        onDraftChange={onDraftChange}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole('combobox', { name: /order type|emir tipi/i }));
    await user.click(await screen.findByText(/^OCO$/));

    fireEvent.click(screen.getByRole('button', { name: /25% \/ 25% \/ 50%|%25 \/ %25 \/ %50/ }));

    const qty = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(qty, { target: { value: '1' } });
    fireEvent.blur(qty);
    const stop = screen.getByRole('spinbutton', { name: /stop loss|zarar kes/i });
    fireEvent.change(stop, { target: { value: '90' } });
    fireEvent.blur(stop);

    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as {
        type: string;
        takeProfitTargets?: { percent: number; offsetPercent?: number }[];
      };
      expect(last.type).toBe('oco');
      expect(last.takeProfitTargets).toEqual([
        { percent: 25, offsetPercent: 5 },
        { percent: 25, offsetPercent: 10 },
        { percent: 50, offsetPercent: 15 },
      ]);
    });

    fireEvent.click(screen.getByRole('button', { name: /place order|emri gönder/i }));
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    const placed = onSubmit.mock.calls.at(-1)?.[0] as {
      orderType: string;
      quantity: number;
      stopLossPrice?: number;
      takeProfitTargets?: { percent: number; offsetPercent?: number }[];
    };
    expect(placed.orderType).toBe('oco');
    expect(placed.quantity).toBe(1);
    expect(placed.stopLossPrice).toBe(90);
    expect(placed.takeProfitTargets).toEqual([
      { percent: 25, offsetPercent: 5 },
      { percent: 25, offsetPercent: 10 },
      { percent: 50, offsetPercent: 15 },
    ]);
    expect(placed).not.toMatchObject({ moveStopToEntry: true });
  });

  it('sends the move-stop option when the box is checked', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onDraftChange = vi.fn();
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        onDraftChange={onDraftChange}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole('combobox', { name: /order type|emir tipi/i }));
    await user.click(await screen.findByText(/^OCO$/));
    await user.click(screen.getByTestId('move-stop-to-entry'));

    const qty = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(qty, { target: { value: '1' } });
    const stop = screen.getByRole('spinbutton', { name: /stop loss|zarar kes/i });
    fireEvent.change(stop, { target: { value: '90' } });
    const gain = screen.getByRole('spinbutton', { name: /gain %|kazanç %/i });
    fireEvent.change(gain, { target: { value: '5' } });

    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as { moveStopToEntry?: boolean };
      expect(last.moveStopToEntry).toBe(true);
    });

    fireEvent.click(screen.getByRole('button', { name: /place order|emri gönder/i }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    const placed = onSubmit.mock.calls.at(-1)?.[0] as { moveStopToEntry?: boolean };
    expect(placed.moveStopToEntry).toBe(true);
  });
});
