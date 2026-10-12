import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { renderWithProviders } from '@/test/render';
import { isCashEquity, openPositionFor } from './helpers';
import { PositionSituation } from './PositionSituation';

const entry = {
  symbol: 'BTCUSDT',
  priceCvd: { kind: 'price_up_cvd_down' as const },
};

const now = {
  symbol: 'BTCUSDT',
  priceCvd: { kind: 'agrees' as const },
};

describe('PositionSituation', () => {
  it('shows the stored entry read beside the live read', () => {
    renderWithProviders(<PositionSituation entry={entry} now={now} />);
    expect(screen.getByRole('group', { name: 'Position read' })).toBeInTheDocument();
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Market when this order filled.')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    expect(screen.getByText('Price and futures CVD are moving together.')).toBeInTheDocument();
  });

  it('uses the Turkish fill label', async () => {
    await i18n.changeLanguage('tr');
    renderWithProviders(<PositionSituation entry={entry} now={now} />);
    expect(screen.getByText('Bu emir dolduğundaki piyasa.')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('keeps the entry line when the live read is still loading', () => {
    renderWithProviders(<PositionSituation entry={entry} nowLoading />);
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('Reading the tape now…')).toBeInTheDocument();
    expect(screen.queryByText('Now')).not.toBeInTheDocument();
  });

  it('renders nothing when neither read has a clause', () => {
    renderWithProviders(<PositionSituation />);
    expect(screen.queryByRole('group', { name: 'Position read' })).not.toBeInTheDocument();
  });

  it('names the clause that changed between the entry and the live read', async () => {
    renderWithProviders(
      <PositionSituation
        entry={entry}
        now={now}
        comparison={{
          same: false,
          clauses: [{ clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' }],
        }}
      />,
    );
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    await i18n.changeLanguage('tr');
    expect(
      screen.getByText('Önce: Fiyat yükselirken vadeli CVD düşüyor. Şimdi: Fiyat ile vadeli CVD aynı yönde.'),
    ).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('names a blank live clause as missing and keeps an unchanged read quiet', () => {
    renderWithProviders(
      <PositionSituation
        entry={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'price_up_cvd_down' },
          funding: { payer: 'long', ratePct: 0.01 },
        }}
        now={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'price_up_cvd_down' },
          fundingMissing: ['binance', 'okx'],
        }}
        comparison={{
          same: false,
          clauses: [{ clause: 'funding', status: 'missing', entry: 'long', missing: ['binance', 'okx'] }],
        }}
      />,
    );
    expect(
      screen.getAllByText('Funding stays blank until Binance and OKX have a rate and open interest.'),
    ).toHaveLength(2);
    expect(screen.queryByText(/Was:/)).not.toBeInTheDocument();
  });

  it('says the entry read still matches', () => {
    renderWithProviders(
      <PositionSituation entry={entry} now={entry} comparison={{ same: true, clauses: [] }} />,
    );
    expect(screen.getByText('The read you entered on is still the live read.')).toBeInTheDocument();
  });
});

describe('openPositionFor', () => {
  it('returns the open row for this coin and skips a flat or equity book', () => {
    const row = openPositionFor(
      [
        { exchange: 'binance', symbol: 'ETHUSDT', quantity: 2 },
        { exchange: 'binance', symbol: 'BTCUSDT', quantity: 0 },
        { exchange: 'binance', symbol: 'BTCUSDT', quantity: 1 },
      ],
      'binance',
      'BTCUSDT',
    );
    expect(row?.quantity).toBe(1);
    expect(openPositionFor([{ exchange: 'nasdaq', symbol: 'AAPL', quantity: 1 }], 'binance', 'AAPL')).toBeUndefined();
    expect(isCashEquity('nasdaq')).toBe(true);
    expect(isCashEquity('binance')).toBe(false);
  });
});
