import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsStatCards } from './OptionsStatCards';

describe('OptionsStatCards', () => {
  it('renders headline options stats', () => {
    renderWithProviders(
      <OptionsStatCards
        data={{
          asset: 'BTC',
          exchange: 'deribit',
          openInterest: '350',
          openInterestUsd: '35000000',
          volume: '12',
          volumeUsd: '1200000',
          putCallOi: '0.4',
          lean: 'calls',
          dvol: '70',
          volCompare: { dvol: '70', realized: '45', gap: '+25', days: 30, lean: 'implied_richer' },
          atm: { iv: '51', strike: '100000', daysToExpiry: 26 },
        }}
      />,
    );
    expect(screen.getByText('35.00M')).toBeInTheDocument();
    expect(screen.getByText('70.0%')).toBeInTheDocument();
    expect(screen.getByText(/RV 45\.0% · gap \+25\.0/)).toBeInTheDocument();
    expect(screen.getByText('0.40')).toBeInTheDocument();
  });
});
