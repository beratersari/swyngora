import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CoinDetailStats } from './CoinDetailStats';

describe('CoinDetailStats', () => {
  it('renders stat tiles', () => {
    render(
      <CoinDetailStats
        items={[
          { label: 'Open', value: '1' },
          { label: 'High', value: '2' },
        ]}
        isLoading={false}
        tickerError={null}
        supplyError={null}
      />,
    );
    expect(screen.getByText('Open')).toBeTruthy();
    expect(screen.getByText('High')).toBeTruthy();
  });

  it('shows ticker and supply errors', () => {
    render(
      <CoinDetailStats
        items={[]}
        isLoading={false}
        tickerError="timeout"
        supplyError="missing"
      />,
    );
    expect(screen.getByText(/Ticker: timeout/)).toBeTruthy();
    expect(screen.getByText(/Supply: missing/)).toBeTruthy();
  });
});
