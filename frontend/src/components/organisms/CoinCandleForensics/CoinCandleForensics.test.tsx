import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { CoinCandleForensics } from './CoinCandleForensics';
import { coinForensicsHref } from './helpers';

const report = {
  note: 'Informational only.',
  why: {
    headline: 'BTC rose on unusually heavy trading.',
    before: 'Bids had already thinned.',
    during: 'Open interest fell as price rose.',
    after: 'After the move, BTC went quiet.',
    similar: 'Similar setups more often kept rising.',
  },
  timeline: {
    summary: 'Open interest printed before price.',
    events: [{ at: '2026-08-20T13:52:00Z', offsetLabel: '-8m', title: 'Binance open interest started up' }],
  },
  similar: {
    summary: '2 similar past setups',
    matches: [
      {
        at: '2026-08-18T12:00:00Z',
        until: '2026-08-18T12:15:00Z',
        direction: 'up',
        similarity: '82',
        coverage: '90',
        afterReturnPct: '+3.5',
        whySimilar: 'Matched most on open interest.',
      },
    ],
  },
};

describe('CoinCandleForensics', () => {
  it('asks for a candle until one is clicked', () => {
    renderWithProviders(
      <CoinCandleForensics timeLabel={null} onClear={vi.fn()} onOpenMove={vi.fn()} />,
    );
    expect(screen.getByTestId('chart-forensics-hint')).toHaveTextContent(/click a candle|bir muma tıklayın/i);
    expect(screen.queryByTestId('chart-forensics')).not.toBeInTheDocument();
  });

  it('shows what changed and what similar setups did next', async () => {
    const onClear = vi.fn();
    const onOpenMove = vi.fn();
    const href = coinForensicsHref('BTCUSDT', '2026-08-20T14:00:00.000Z');
    renderWithProviders(
      <CoinCandleForensics
        timeLabel="2026-08-20 14:00"
        report={report}
        fullReadTo={href}
        onClear={onClear}
        onOpenMove={onOpenMove}
      />,
    );
    expect(screen.getByTestId('chart-forensics')).toHaveTextContent('2026-08-20 14:00');
    expect(screen.getByText(/BTC rose on unusually heavy trading/i)).toBeInTheDocument();
    expect(screen.getByText(/open interest started up/i)).toBeInTheDocument();
    expect(screen.getByText(/2 similar past setups/i)).toBeInTheDocument();
    expect(screen.getByText('+3.5')).toBeInTheDocument();
    expect(screen.getByTestId('chart-forensics-full')).toHaveAttribute('href', href);
    await userEvent.click(screen.getByTestId('chart-forensics-clear'));
    expect(onClear).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByText('+3.5'));
    expect(onOpenMove).toHaveBeenCalledWith('2026-08-18T12:00:00.000Z');
  });

  it('shows a load failure without the previous story', () => {
    renderWithProviders(
      <CoinCandleForensics
        timeLabel="2026-08-20 14:00"
        report={report}
        error="Forensics could not be loaded."
        onClear={vi.fn()}
        onOpenMove={vi.fn()}
      />,
    );
    expect(screen.getByTestId('chart-forensics-error')).toHaveTextContent('Forensics could not be loaded.');
    expect(screen.queryByText(/BTC rose on unusually heavy trading/i)).not.toBeInTheDocument();
  });
});
