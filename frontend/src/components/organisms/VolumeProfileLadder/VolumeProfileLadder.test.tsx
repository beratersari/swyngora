import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { VolumeProfileLadder } from './VolumeProfileLadder';

describe('VolumeProfileLadder', () => {
  it('shows volume, open interest, and liquidations on the same price row', () => {
    renderWithProviders(
      <VolumeProfileLadder
        report={{
          summary: 'BTC volume clustered at 65200.',
          combined: {
            exchange: 'all',
            oiKnown: true,
            liqKnown: true,
            bins: [
              {
                price: '65200',
                high: '65300',
                volume: '150000',
                buyVolume: '90000',
                sellVolume: '60000',
                oiOpened: '300000',
                oiClosed: '50000',
                longLiq: '80000',
                shortLiq: '20000',
                isPoc: true,
                inValueArea: true,
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText(/65[,.]200/)).toBeInTheDocument();
    expect(screen.getByText('300.00K / 50.00K')).toBeInTheDocument();
    expect(screen.getByText('80.00K / 20.00K')).toBeInTheDocument();
    expect(screen.getByRole('listitem').getAttribute('title')).toContain('oi +300.00K');
    expect(screen.getByRole('listitem').getAttribute('title')).toContain('liq L 80.00K');
    expect(screen.getByText('Open interest')).toBeInTheDocument();
    expect(screen.getByText('Liquidations')).toBeInTheDocument();
  });
});
