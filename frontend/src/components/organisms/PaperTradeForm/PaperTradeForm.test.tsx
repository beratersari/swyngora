import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
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
    const spin = screen.getByRole('spinbutton');
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
});
