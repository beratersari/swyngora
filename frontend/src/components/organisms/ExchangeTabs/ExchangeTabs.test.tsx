import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithTheme } from '@/test/render';
import { ExchangeTabs } from './ExchangeTabs';

describe('ExchangeTabs', () => {
  it('shows skeleton when loading with no exchanges', () => {
    const { container } = renderWithTheme(
      <ExchangeTabs exchanges={[]} value="binance" onChange={() => undefined} isLoading />,
    );
    expect(container.querySelector('.ant-skeleton')).toBeTruthy();
  });

  it('renders exchange tabs and fires onChange', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderWithTheme(
      <ExchangeTabs
        exchanges={['binance', 'coinbase']}
        value="binance"
        onChange={onChange}
      />,
    );
    await user.click(screen.getByRole('tab', { name: /coinbase/i }));
    expect(onChange).toHaveBeenCalledWith('coinbase');
  });
});
