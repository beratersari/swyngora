import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { LiquidationWindowCards } from './LiquidationWindowCards';

describe('LiquidationWindowCards', () => {
  it('renders total / long / short and selects a window', async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <LiquidationWindowCards
        selectedWindow="24h"
        onSelect={onSelect}
        windows={[
          { window: '1h', totalNotional: '1000000', longNotional: '700000', shortNotional: '300000', count: 4, complete: true },
          { window: '4h', totalNotional: '2000000', longNotional: '1000000', shortNotional: '1000000', count: 8 },
          { window: '12h', totalNotional: '3000000', longNotional: '1200000', shortNotional: '1800000', count: 12 },
          { window: '24h', totalNotional: '4000000', longNotional: '1500000', shortNotional: '2500000', count: 20 },
        ]}
      />,
    );
    expect(screen.getAllByText(/total|toplam/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/long/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/short/i).length).toBeGreaterThan(0);
    await user.click(screen.getByRole('button', { name: /1h|1s/i }));
    expect(onSelect).toHaveBeenCalledWith('1h');
  });
});
