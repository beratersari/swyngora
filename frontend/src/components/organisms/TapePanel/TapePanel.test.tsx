import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { TapePanel } from './TapePanel';

describe('TapePanel', () => {
  it('renders OI, funding, liquidations, and CVD', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          venueCount: 2,
          current: { value: '1000000000' },
          windows: [{ window: '1h', changeValuePct: '1.2', complete: true }],
          funding: { current: { ratePct: '0.01', payer: 'long' } },
        }}
        liquidations={{
          windows: [{ window: '1h', totalNotional: '25000000', complete: true }],
        }}
        cvd={{ combined: { lastCvd: '12.5', summary: 'buy flow' } }}
      />,
    );
    expect(screen.getByText('Futures tape')).toBeInTheDocument();
    expect(screen.getByText('1.00B')).toBeInTheDocument();
    expect(screen.getByText('0.0100%')).toBeInTheDocument();
    expect(screen.getByText('25.00M')).toBeInTheDocument();
    expect(screen.getByText('buy flow')).toBeInTheDocument();
  });

  it('explains a missing perp book', () => {
    renderWithTheme(<TapePanel openInterestError="no perpetual on this venue" />);
    expect(screen.getAllByText('no perpetual on this venue').length).toBeGreaterThan(0);
  });
});
