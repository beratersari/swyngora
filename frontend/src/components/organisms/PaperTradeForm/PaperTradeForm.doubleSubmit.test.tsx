import { describe, expect, it, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PaperTradeForm } from './PaperTradeForm';

describe('PaperTradeForm double-submit guard', () => {
  it('CRITICAL: only one market buy when Buy is clicked twice before resolve', async () => {
    let resolveSubmit: (() => void) | undefined;
    const onSubmit = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          resolveSubmit = resolve;
        }),
    );

    renderWithTheme(
      <PaperTradeForm
        lockedExchange="binance"
        lockedSymbol="BTCUSDT"
        compact
        isSubmitting={false}
        onSubmit={onSubmit}
      />,
    );

    const spin = screen.getByRole('spinbutton');
    fireEvent.change(spin, { target: { value: '0.01' } });
    fireEvent.blur(spin);

    const buy = screen.getByRole('button', { name: /paper buy|kağıt al/i });
    fireEvent.click(buy);
    fireEvent.click(buy);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalled();
    });

    expect(onSubmit).toHaveBeenCalledTimes(1);

    resolveSubmit?.();
    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledTimes(1);
    });
  });
});
