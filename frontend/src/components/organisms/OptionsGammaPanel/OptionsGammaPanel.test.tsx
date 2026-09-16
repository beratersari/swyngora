import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { OptionsGammaPanel } from './OptionsGammaPanel';

describe('OptionsGammaPanel', () => {
  it('shows net GEX, regime, and the flip', () => {
    renderWithProviders(
      <OptionsGammaPanel
        data={{
          asset: 'BTC',
          indexPrice: '100000',
          gamma: {
            netUsd: '-840000000',
            callUsd: '-1200000000',
            putUsd: '360000000',
            regime: 'short',
            flip: '97200',
            flipSide: 'above',
            flipDistPct: '2.8',
            used: 40,
            profile: [
              { spot: '90000', netUsd: '200000000' },
              { spot: '97200', netUsd: '-10000000' },
              { spot: '110000', netUsd: '-400000000' },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText(/net dealer gex/i)).toBeInTheDocument();
    expect(screen.getByText('−840.00M')).toBeInTheDocument();
    expect(screen.getByText(/moves can run faster/i)).toBeInTheDocument();
    expect(screen.getByText(/97[,.]200/)).toBeInTheDocument();
  });
});
