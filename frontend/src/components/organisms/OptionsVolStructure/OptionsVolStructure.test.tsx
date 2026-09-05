import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsVolStructure } from './OptionsVolStructure';

describe('OptionsVolStructure', () => {
  it('renders term slope and ATM skew', () => {
    renderWithProviders(
      <OptionsVolStructure
        data={{
          termStructure: [
            { label: '27 Mar 2026', expiry: '2026-03-27T08:00:00Z', atmIv: '51', callIv: '50', putIv: '52', skew: '2' },
            { label: '26 Jun 2026', expiry: '2026-06-26T08:00:00Z', atmIv: '48', callIv: '48', putIv: '48', skew: '0' },
          ],
          skew: { atmSkew: '2', termSlope: '-3', lean: 'puts', frontDte: 26, backDte: 117, wingSkew: '4' },
        }}
      />,
    );
    expect(screen.getAllByText('+2.0').length).toBeGreaterThan(0);
    expect(screen.getByText('-3.0')).toBeInTheDocument();
    expect(screen.getAllByText('27 Mar 2026').length).toBeGreaterThan(0);
  });
});
