import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsExpiryDetail } from './OptionsExpiryDetail';

describe('OptionsExpiryDetail', () => {
  it('shows expected move and max pain', () => {
    renderWithProviders(
      <OptionsExpiryDetail
        indexPrice="100000"
        expiry={{
          label: '27 Mar 2026',
          daysToExpiry: 26,
          expectedMovePct: '13.6',
          expectedMoveAbs: '13600',
          expectedLow: '86400',
          expectedHigh: '113600',
          maxPain: '100000',
          strikes: [
            { strike: '90000', payout: '200000', callOi: '0', putOi: '10' },
            { strike: '100000', payout: '0', callOi: '10', putOi: '10', isMaxPain: true },
          ],
        }}
      />,
    );
    expect(screen.getByText('±13.6%')).toBeInTheDocument();
    expect(screen.getByText(/lowest payout/i)).toBeInTheDocument();
    expect(screen.getAllByText(/100[.,]000/).length).toBeGreaterThan(0);
  });
});
