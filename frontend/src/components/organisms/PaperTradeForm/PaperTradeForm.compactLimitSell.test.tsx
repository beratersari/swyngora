import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { PaperTradeForm } from './PaperTradeForm';

describe('PaperTradeForm compact limit sell', () => {
  it('lets the user place a limit sell from the coin-detail compact ticket', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        advanced={false}
        onSubmit={onSubmit}
      />,
    );

    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByTitle(/^Limit$/i));

    const qty = screen.getByRole('spinbutton', { name: /quantity|miktar/i });
    await user.clear(qty);
    await user.type(qty, '1');
    const trigger = screen.getByRole('spinbutton', { name: /trigger|fiyat|eşik/i });
    await user.clear(trigger);
    await user.type(trigger, '60000');

    // Compact detail ticket must expose a sell path (button or side control).
    const sellControl =
      screen.queryByRole('button', { name: /paper sell|kağıt sat|sell/i }) ??
      screen.queryByRole('radio', { name: /sell|sat/i });
    expect(sellControl).not.toBeNull();
    await user.click(sellControl!);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });
    expect(onSubmit.mock.calls[0][0]).toMatchObject({
      orderType: 'limit_sell',
      side: 'sell',
      quantity: 1,
      triggerPrice: 60000,
    });
  });
});
