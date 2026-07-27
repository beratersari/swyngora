import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CandleChart } from './CandleChart';

describe('CandleChart', () => {
  it('shows loading skeleton', () => {
    const { container } = render(
      <CandleChart candles={[]} isLoading errorMessage={null} />,
    );
    expect(container.firstChild).toBeTruthy();
  });

  it('shows error message', () => {
    render(
      <CandleChart
        candles={[]}
        isLoading={false}
        errorMessage="Candle fetch failed"
      />,
    );
    expect(screen.getByText('Candle fetch failed')).toBeTruthy();
  });

  it('shows empty message', () => {
    render(
      <CandleChart candles={[]} isLoading={false} errorMessage={null} />,
    );
    expect(screen.getByText('No candle data')).toBeTruthy();
  });
});
