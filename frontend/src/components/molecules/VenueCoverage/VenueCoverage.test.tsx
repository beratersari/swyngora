import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { VenueCoverage } from './VenueCoverage';

describe('VenueCoverage', () => {
  it('names a missing venue and an old venue and withholds a market read', () => {
    renderWithTheme(
        <VenueCoverage
          coverage={{
            complete: false,
            venues: [
              { exchange: 'binance', status: 'fresh', ageSec: 10 },
              { exchange: 'bybit', status: 'stale', ageSec: 360 },
              { exchange: 'okx', status: 'missing', ageSec: 0 },
            ],
            missing: ['okx'],
            stale: ['bybit'],
          }}
        />,
    );
    expect(screen.getByTestId('venue-coverage')).toHaveTextContent('No market read until every venue is fresh.');
    expect(screen.getByTestId('venue-coverage')).toHaveTextContent('Bybit: old (6m old)');
    expect(screen.getByTestId('venue-coverage')).toHaveTextContent('OKX: missing');
    expect(screen.getByTestId('venue-coverage')).toHaveTextContent('Binance: fresh');
  });
});
