import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { BasisPanel } from './BasisPanel';

describe('BasisPanel', () => {
  it('shows perp gap and near vs far curve', () => {
    renderWithProviders(
      <BasisPanel
        report={{
          symbol: 'BTCUSDT',
          agreement: { title: 'Binance and Bybit show the same kind of gap', alignment: 'same' },
          venues: [
            {
              exchange: 'binance',
              last: { kind: 'premium', deltaPct: '+0.12' },
              trend: 'expanding',
              summary: 'Binance: perpetual is a premium.',
              curve: {
                complete: true,
                shape: 'contango',
                slope: 'up',
                richer: 'far',
                spreadPct: '+0.62',
                summary: 'Far expiry (26 Dec 2026) is more expensive than near (26 Sep 2026).',
                near: { label: '26 Sep 2026', mark: '64200' },
                far: { label: '26 Dec 2026', mark: '64600' },
                points: [
                  { symbol: 'BTCUSDT', kind: 'perp', label: 'Perp', mark: '64000', vsIndexPct: '+0.00' },
                  { symbol: 'BTCUSDT_260925', kind: 'current_quarter', label: '26 Sep 2026', daysToExpiry: 42, mark: '64200', vsIndexPct: '+0.31' },
                  { symbol: 'BTCUSDT_261225', kind: 'next_quarter', label: '26 Dec 2026', daysToExpiry: 133, mark: '64600', vsIndexPct: '+0.94' },
                ],
              },
            },
          ],
        }}
      />,
    );
    expect(screen.getByTestId('basis-panel')).toBeInTheDocument();
    expect(screen.getByText(/same kind of gap/i)).toBeInTheDocument();
    expect(screen.getAllByText(/far expiry is more expensive/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/contango/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/curve up/i).length).toBeGreaterThan(0);
    expect(screen.getByText('Perp')).toBeInTheDocument();
    expect(screen.getAllByText('26 Dec 2026').length).toBeGreaterThan(0);
  });
});
