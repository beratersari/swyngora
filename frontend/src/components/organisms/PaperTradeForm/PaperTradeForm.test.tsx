import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PaperTradeForm } from './PaperTradeForm';

describe('PaperTradeForm', () => {
  it('submits market buy with quantity', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onSubmit={onSubmit}
      />,
    );
    const spin = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(spin, { target: { value: '0.01' } });
    // Ant Design InputNumber may need blur to commit
    fireEvent.blur(spin);
    fireEvent.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    const arg = onSubmit.mock.calls[0][0] as {
      exchange: string;
      symbol: string;
      side: string;
      quantity: number;
    };
    expect(arg.exchange).toBe('binance');
    expect(arg.symbol).toBe('BTCUSDT');
    expect(arg.side).toBe('buy');
    expect(arg.orderType).toBe('market');
    expect(arg.quantity).toBe(0.01);
  });

  it('sends a best-cost market order across venues', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const onDraftChange = vi.fn();
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onDraftChange={onDraftChange}
        onSubmit={onSubmit}
      />,
    );
    fireEvent.click(screen.getByText(/best cost|en düşük maliyet/i));
    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as {
        exchange: string;
        compareExchange?: string;
      };
      expect(last.exchange).toBe('all');
      expect(last.compareExchange).toBe('binance');
    });
    const spin = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(spin, { target: { value: '0.5' } });
    fireEvent.blur(spin);
    const slip = screen.getByRole('spinbutton', { name: /max slippage|kayma/i });
    fireEvent.change(slip, { target: { value: '0.5' } });
    fireEvent.blur(slip);
    fireEvent.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    const arg = onSubmit.mock.calls.at(-1)?.[0] as {
      exchange: string;
      quantity: number;
      maxSlippagePct?: number;
    };
    expect(arg.exchange).toBe('all');
    expect(arg.quantity).toBe(0.5);
    expect(arg.maxSlippagePct).toBe(0.5);
  });

  it('finding 8: limit ticket posts the raw typed trigger with no FX conversion', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PaperTradeForm lockedExchange="binance" lockedSymbol="BTCUSDT" onSubmit={onSubmit} />,
    );
    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByTitle(/^Limit$/i));
    const qty = screen.getByLabelText(/quantity|miktar/i);
    const trigger = screen.getByLabelText(/trigger|fiyat|eşik/i);
    await user.clear(qty);
    await user.type(qty, '1');
    await user.clear(trigger);
    await user.type(trigger, '3400000');
    await user.click(screen.getByRole('button', { name: /paper buy|kağıt al/i }));
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    const arg = onSubmit.mock.calls[0][0] as { triggerPrice?: number; quantity: number };
    expect(arg.quantity).toBe(1);
    expect(arg.triggerPrice).toBe(3400000);
  });

  it('finding 8: does not prefill a converted last into the ticket', () => {
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onSubmit={async () => undefined}
      />,
    );
    const spin = screen.getByRole('spinbutton', { name: /quantity|miktar/i }) as HTMLInputElement;
    expect(spin.value === '' || spin.value === '0').toBe(true);
  });

  it('reports the draft so the page can price it', async () => {
    const onDraftChange = vi.fn();
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onSubmit={async () => undefined}
        onDraftChange={onDraftChange}
      />,
    );
    const spin = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    fireEvent.change(spin, { target: { value: '2' } });
    fireEvent.blur(spin);
    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as {
        quantity: number;
        side: string;
        type: string;
      };
      expect(last.quantity).toBe(2);
      expect(last.side).toBe('buy');
      expect(last.type).toBe('market');
    });
    fireEvent.pointerDown(screen.getByRole('button', { name: /paper sell|kağıt sat/i }));
    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as { side: string };
      expect(last.side).toBe('sell');
    });
  });

  it('sizes the buy from a risk percent and the stop', async () => {
    const onDraftChange = vi.fn();
    const { rerender } = renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onSubmit={async () => undefined}
        onDraftChange={onDraftChange}
      />,
    );
    fireEvent.change(screen.getByRole('spinbutton', { name: /risk/i }), { target: { value: '1' } });
    fireEvent.blur(screen.getByRole('spinbutton', { name: /risk/i }));
    fireEvent.change(screen.getByRole('spinbutton', { name: /stop loss|zarar kes/i }), {
      target: { value: '90' },
    });
    fireEvent.blur(screen.getByRole('spinbutton', { name: /stop loss|zarar kes/i }));
    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as {
        riskPercent?: number;
        stopLossPrice?: number;
      };
      expect(last.riskPercent).toBe(1);
      expect(last.stopLossPrice).toBe(90);
    });
    rerender(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        onSubmit={async () => undefined}
        onDraftChange={onDraftChange}
        preview={{
          product: 'spot',
          type: 'market',
          side: 'buy',
          symbol: 'BTCUSDT',
          quantity: 0.25,
          fillsNow: true,
          sizing: {
            riskPercent: 1,
            riskBudget: 100,
            estimatedLoss: 100,
            quantity: 0.25,
            stopAttached: false,
          },
        }}
      />,
    );
    await waitFor(() => {
      const qty = screen.getByRole('spinbutton', { name: /quantity|miktar/i }) as HTMLInputElement;
      expect(Number(qty.value)).toBe(0.25);
    });
    await waitFor(() => {
      const last = onDraftChange.mock.calls.at(-1)?.[0] as {
        quantity: number;
        riskPercent?: number;
      };
      expect(last.quantity).toBe(0.25);
      expect(last.riskPercent).toBe(1);
    });
  });
});
