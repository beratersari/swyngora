import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PumpHitRow } from './PumpHitRow';

describe('PumpHitRow', () => {
  it('renders and presses', () => {
    const onPress = vi.fn();
    render(
      <PumpHitRow
        row={{
          id: 'binance|BTCUSDT',
          symbol: 'BTCUSDT',
          exchange: 'binance',
          bestReturnLabel: '+10.00%',
          bestReturnTone: 'success',
          eventsLabel: '1 event',
          metaLabel: '15m',
        }}
        onPress={onPress}
      />,
    );
    fireEvent.click(screen.getByText('BTCUSDT'));
    expect(onPress).toHaveBeenCalledWith('binance', 'BTCUSDT');
  });
});
