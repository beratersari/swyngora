import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ExchangeChips } from './ExchangeChips';

describe('ExchangeChips', () => {
  it('renders exchanges and selects', () => {
    const onSelect = vi.fn();
    render(
      <ExchangeChips
        exchanges={['binance', 'coinbase']}
        selected="binance"
        onSelect={onSelect}
      />,
    );
    expect(screen.getByText(/binance/i)).toBeTruthy();
    fireEvent.click(screen.getByText(/coinbase/i));
    expect(onSelect).toHaveBeenCalled();
  });
});
