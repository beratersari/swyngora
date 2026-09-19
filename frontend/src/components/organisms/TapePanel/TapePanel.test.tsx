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
        snapshot={{
          summary: 'Volume building ahead of price',
          combined: { longPct: '55.2', windows: [{ window: '1h', taker: { dominant: 'buy', buyShare: '0.58' } }] },
        }}
      />,
    );
    expect(screen.getByText('Futures tape')).toBeInTheDocument();
    expect(screen.getByText('1.00B')).toBeInTheDocument();
    expect(screen.getByText('0.0100%')).toBeInTheDocument();
    expect(screen.getByText('25.00M')).toBeInTheDocument();
    expect(screen.getByText('buy flow')).toBeInTheDocument();
    expect(screen.getByText('Volume building ahead of price')).toBeInTheDocument();
  });

  it('shows OI-weighted funding pressure and window change', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          funding: {
            pressure: {
              ratePct: '0.007778',
              payer: 'long',
              summary: 'Binance 88.9% of OI at 0.01%. Combined pressure 0.007778% (longs pay).',
            },
            pressureWindows: [
              { window: '1h', changePct: '+0.001', complete: true },
              { window: '24h', changePct: '-0.002', complete: true },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText('0.0078%')).toBeInTheDocument();
    expect(screen.getByText('+0.0010%')).toBeInTheDocument();
    expect(screen.getByText('-0.0020%')).toBeInTheDocument();
    expect(screen.getByText(/88.9%/)).toBeInTheDocument();
  });

  it('shows top-trader size and a crowd-vs-size split', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          longShort: {
            venues: [
              {
                exchange: 'binance',
                current: { longPct: '68.0', shortPct: '32.0', bias: 'long' },
                topPositions: { longPct: '34.0', shortPct: '66.0', bias: 'short' },
                split: {
                  alignment: 'opposite',
                  important: true,
                  headline: 'Crowd accounts and top-trader positions lean opposite.',
                  detail: 'Size is with the bigger books, not the crowd.',
                },
              },
            ],
          },
        }}
        snapshot={{ combined: { longPct: '68.0', windows: [] } }}
      />,
    );
    expect(screen.getByText('34.0%')).toBeInTheDocument();
    expect(screen.getByText(/lean opposite/i)).toBeInTheDocument();
  });

  it('explains a missing perp book', () => {
    renderWithTheme(<TapePanel openInterestError="no perpetual on this venue" />);
    expect(screen.getAllByText('no perpetual on this venue').length).toBeGreaterThan(0);
  });
});
