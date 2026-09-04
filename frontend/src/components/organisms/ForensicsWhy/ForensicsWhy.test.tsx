import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { ForensicsWhy } from './ForensicsWhy';

describe('ForensicsWhy', () => {
  it('renders the story sections', () => {
    renderWithProviders(
      <ForensicsWhy
        why={{
          headline: 'BTC rose on unusually heavy trading.',
          before: 'Bids had already thinned, so there was less support under the market.',
          during: 'Open interest fell as price rose — a short-cover shape.',
          after: 'After the move, BTC went quiet.',
          similar: 'When the tape looked like this before, BTC more often kept rising over the next 1h.',
          caution: 'This is a read of the tape, not a prediction or financial advice.',
        }}
      />,
    );
    expect(screen.getByText(/why this may have happened|neden olmuş olabilir/i)).toBeInTheDocument();
    expect(screen.getByText(/BTC rose on unusually heavy trading/i)).toBeInTheDocument();
    expect(screen.getByText(/less support under the market/i)).toBeInTheDocument();
    expect(screen.getByText(/short-cover/i)).toBeInTheDocument();
    expect(screen.getByText(/more often kept rising/i)).toBeInTheDocument();
  });
});
