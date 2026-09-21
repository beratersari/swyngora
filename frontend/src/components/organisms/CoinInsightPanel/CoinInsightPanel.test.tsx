import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { CoinInsightPanel } from './CoinInsightPanel';

describe('CoinInsightPanel', () => {
  it('renders title, stats, and summary', () => {
    renderWithTheme(
      <CoinInsightPanel
        title="Taker flow"
        summary="Buyers are lifting the offer."
        stats={[{ label: '1h', value: 'buy' }]}
      />,
    );
    expect(screen.getByText('Taker flow')).toBeInTheDocument();
    expect(screen.getByText('Buyers are lifting the offer.')).toBeInTheDocument();
    expect(screen.getByText('buy')).toBeInTheDocument();
  });
});
