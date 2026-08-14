import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PortfolioOrdersTable } from './PortfolioOrdersTable';
import type { PendingOrder } from '@/libs/api';

const openLimit: PendingOrder = {
  id: 'ord-1',
  exchange: 'binance',
  symbol: 'BTCUSDT',
  type: 'limit_buy',
  side: 'buy',
  status: 'open',
  timeInForce: 'gtc',
  triggerPrice: 90,
  remainingQuantity: 1,
  filledQuantity: 0,
};

describe('PortfolioOrdersTable amend payload', () => {
  it('does not send snapshotted remaining when the user only changes trigger', async () => {
    const onAmend = vi.fn().mockResolvedValue(undefined);
    renderWithTheme(
      <PortfolioOrdersTable items={[openLimit]} onAmend={onAmend} />,
    );

    fireEvent.click(screen.getByRole('button', { name: /amend|düzelt/i }));
    const spins = await screen.findAllByRole('spinbutton');
    fireEvent.change(spins[0], { target: { value: '88' } });
    fireEvent.blur(spins[0]);

    const amendButtons = screen.getAllByRole('button', { name: /amend|düzelt/i });
    fireEvent.click(amendButtons[amendButtons.length - 1]);
    await waitFor(() => {
      expect(onAmend).toHaveBeenCalled();
    });
    const arg = onAmend.mock.calls[0][0] as {
      id: string;
      triggerPrice?: number;
      remainingQuantity?: number;
    };
    expect(arg.id).toBe('ord-1');
    expect(arg.remainingQuantity).toBeUndefined();
  });
});
