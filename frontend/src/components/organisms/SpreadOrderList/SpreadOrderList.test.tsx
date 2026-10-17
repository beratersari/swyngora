import { describe, expect, it, vi } from 'vitest';
import { fireEvent, screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { SpreadOrderList } from './SpreadOrderList';

describe('SpreadOrderList', () => {
  it('shows progress and cancels the rest', () => {
    const onCancel = vi.fn();
    renderWithTheme(
      <SpreadOrderList
        items={[
          {
            id: 's1',
            symbol: 'BTCUSDT',
            side: 'buy',
            quantity: 1,
            filledQuantity: 0.2,
            sliceCount: 15,
            slicesDone: 2,
            durationMinutes: 30,
          },
        ]}
        onCancel={onCancel}
      />,
    );
    expect(screen.getByTestId('spread-order')).toHaveTextContent(/BTCUSDT/);
    expect(screen.getByTestId('spread-order')).toHaveTextContent(/2/);
    fireEvent.click(screen.getByRole('button', { name: /cancel the rest|kalanı iptal/i }));
    expect(onCancel).toHaveBeenCalledWith('s1');
  });

  it('renders nothing when no spread is working', () => {
    const { container } = renderWithTheme(<SpreadOrderList items={[]} onCancel={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });
});
