import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsExpiryBars } from './OptionsExpiryBars';

describe('OptionsExpiryBars', () => {
  it('lists expiries', () => {
    renderWithProviders(
      <OptionsExpiryBars
        expiries={[
          { label: '27 Mar 2026', expiry: '2026-03-27T08:00:00Z', sharePct: '80', callOi: '8', putOi: '2', openInterestUsd: '8000000' },
        ]}
      />,
    );
    expect(screen.getByText('27 Mar 2026')).toBeInTheDocument();
    expect(screen.getByText('8.00M')).toBeInTheDocument();
  });
});
