import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithTheme } from '@/test/render';
import { PositionSidePanel } from './PositionSidePanel';

describe('PositionSidePanel', () => {
  it('shows net long and net short for each window', () => {
    renderWithTheme(
      <PositionSidePanel
        data={{
          symbol: 'BTCUSDT',
          windows: [
            { window: '15m', netLong: '45000000', netShort: '-15000000', complete: true },
            { window: '1h', netLong: '10000000', netShort: '4000000', complete: true },
            { window: '4h', netLong: '-8000000', netShort: '2000000', complete: true },
          ],
        }}
      />,
    );
    expect(screen.getByText('Net long / short')).toBeInTheDocument();
    expect(screen.getByText('Long +45.00M · Short -15.00M')).toBeInTheDocument();
    expect(screen.getByText('Long +10.00M · Short +4.00M')).toBeInTheDocument();
    expect(screen.getByText('Long -8.00M · Short +2.00M')).toBeInTheDocument();
  });

  it('blanks an incomplete window and names the venue', () => {
    renderWithTheme(
      <PositionSidePanel
        data={{
          symbol: 'BTCUSDT',
          windows: [
            {
              window: '15m',
              netLong: '45000000',
              netShort: '-15000000',
              complete: false,
              missing: ['bybit'],
            },
            {
              window: '1h',
              netLong: '10000000',
              netShort: '4000000',
              complete: false,
              summary: 'Bybit has no fresh open-interest sample.',
            },
            { window: '4h', netLong: '-8000000', netShort: '2000000', complete: false },
          ],
        }}
      />,
    );
    expect(screen.queryByText(/Long \+45/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Long \+10/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Long -8/)).not.toBeInTheDocument();
    expect(screen.getByText(/until bybit is in/i)).toBeInTheDocument();
    expect(screen.getByText('Bybit has no fresh open-interest sample.')).toBeInTheDocument();
    expect(screen.getByText('Window is incomplete — history is still filling.')).toBeInTheDocument();
  });
});
