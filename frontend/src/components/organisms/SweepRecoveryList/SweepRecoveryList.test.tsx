import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { SweepRecoveryList } from './SweepRecoveryList';

describe('SweepRecoveryList', () => {
  it('shows how long the swept side took to refill', () => {
    renderWithProviders(
      <SweepRecoveryList
        report={{
          rows: [
            { exchange: 'binance', side: 'high', level: '65000', recovered: true, recoverySeconds: 300 },
            { exchange: 'bybit', side: 'low', level: '64000', recovered: false, recoverySeconds: 0 },
          ],
        }}
      />,
    );
    expect(screen.getByText(/65000/)).toHaveTextContent('5m');
    expect(screen.getByText(/64000/)).toBeInTheDocument();
  });
});
