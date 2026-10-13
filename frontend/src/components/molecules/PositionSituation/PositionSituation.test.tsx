import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { formatDateTime } from '@/libs/utils/formatMarket';
import { formatPrice } from '@/libs/utils/formatPrice';
import { renderWithProviders } from '@/test/render';
import { isCashEquity, openPositionFor, positionLotReads } from './helpers';
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
          clauses: [
            { clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' },
          ],
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
      screen.getByText(
        'Önce: Fiyat yükselirken vadeli CVD düşüyor. Şimdi: Fiyat ile vadeli CVD aynı yönde.',
      ),
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
          clauses: [
            { clause: 'funding', status: 'missing', entry: 'long', missing: ['binance', 'okx'] },
          ],
        }}
      />,
    );
    expect(
      screen.getAllByText(
        'Funding stays blank until Binance and OKX have a rate and open interest.',
      ),
    ).toHaveLength(2);
    expect(screen.queryByText(/Was:/)).not.toBeInTheDocument();
  });

  it('says how long the live clause has held and leaves the entry and the change line without a clock', async () => {
    await i18n.changeLanguage('en');
    const since = '2026-10-12T11:30:00Z';
    const time = formatDateTime(since, i18n.language);
    renderWithProviders(
      <PositionSituation
        entry={{ symbol: 'BTCUSDT', priceCvd: { kind: 'price_up_cvd_down', since } }}
        now={{ symbol: 'BTCUSDT', priceCvd: { kind: 'agrees', since } }}
        comparison={{
          same: false,
          clauses: [
            { clause: 'priceCvd', status: 'changed', entry: 'price_up_cvd_down', live: 'agrees' },
          ],
        }}
      />,
    );
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(
      screen.getByText(`Price and futures CVD are moving together. Since ${time}.`),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'Was: Price is rising while futures CVD is falling. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    await i18n.changeLanguage('tr');
    const trTime = formatDateTime(since, 'tr');
    expect(screen.getByText('Fiyat yükselirken vadeli CVD düşüyor.')).toBeInTheDocument();
    expect(
      screen.getByText(`Fiyat ile vadeli CVD aynı yönde. ${trTime} tarihinden beri.`),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'Önce: Fiyat yükselirken vadeli CVD düşüyor. Şimdi: Fiyat ile vadeli CVD aynı yönde.',
      ),
    ).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('says the entry read still matches', () => {
    renderWithProviders(
      <PositionSituation entry={entry} now={entry} comparison={{ same: true, clauses: [] }} />,
    );
    expect(screen.getByText('The read you entered on is still the live read.')).toBeInTheDocument();
  });

  it('lists every remaining buy with its size, time, and what the live read left', async () => {
    await i18n.changeLanguage('en');
    const newer = '2026-10-12T12:00:00Z';
    const older = '2026-10-12T10:00:00Z';
    renderWithProviders(
      <PositionSituation
        now={now}
        lots={[
          {
            id: 'new',
            quantity: 0.4,
            openedAt: newer,
            newest: true,
            situation: { symbol: 'BTCUSDT', priceCvd: { kind: 'price_down_cvd_up' } },
            comparison: {
              same: false,
              clauses: [
                {
                  clause: 'priceCvd',
                  status: 'changed',
                  entry: 'price_down_cvd_up',
                  live: 'agrees',
                },
              ],
            },
          },
          {
            id: 'old',
            quantity: 1,
            openedAt: older,
            situation: entry,
            comparison: {
              same: false,
              clauses: [
                {
                  clause: 'priceCvd',
                  status: 'changed',
                  entry: 'price_up_cvd_down',
                  live: 'agrees',
                },
              ],
            },
          },
          {
            id: 'blank',
            quantity: 0.2,
            openedAt: '2026-10-12T09:00:00Z',
          },
        ]}
      />,
    );
    expect(screen.getByText('Entered as')).toBeInTheDocument();
    expect(screen.getByText('The situation alert follows this buy.')).toBeInTheDocument();
    expect(
      screen.getByText(`${formatPrice(0.4)} remaining, filled ${formatDateTime(newer, 'en')}`),
    ).toBeInTheDocument();
    expect(screen.getAllByText('Earlier buy')).toHaveLength(2);
    expect(
      screen.getByText(`${formatPrice(1)} remaining, filled ${formatDateTime(older, 'en')}`),
    ).toBeInTheDocument();
    expect(screen.getByText('This buy stored no read.')).toBeInTheDocument();
    expect(screen.getByText('Price is falling while futures CVD is rising.')).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Was: Price is falling while futures CVD is rising. Now: Price and futures CVD are moving together.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('Now')).toBeInTheDocument();
    await i18n.changeLanguage('tr');
    expect(screen.getByText('Girişte')).toBeInTheDocument();
    expect(screen.getByText('Durum uyarısı bu alımı izler.')).toBeInTheDocument();
    expect(screen.getAllByText('Önceki alım')).toHaveLength(2);
    expect(screen.getByText('Bu alım okuma saklamadı.')).toBeInTheDocument();
    expect(
      screen.getByText(`Kalan ${formatPrice(0.4)}, dolum ${formatDateTime(newer, 'tr')}`),
    ).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });
});

describe('positionLotReads', () => {
  it('prefers the live comparisons and drops them when the live read failed', () => {
    const stored = [{ id: 'old', quantity: 1, situation: entry }];
    const live = [
      { id: 'old', quantity: 1, situation: entry, comparison: { same: true, clauses: [] } },
    ];
    expect(positionLotReads({ openReads: stored }, { liveOk: true, openReads: live })).toEqual(
      live,
    );
    expect(
      positionLotReads({ openReads: stored }, { liveOk: false, openReads: live })?.[0].comparison,
    ).toBeUndefined();
    expect(positionLotReads({ openReads: stored }, undefined)).toEqual(stored);
    expect(
      positionLotReads({ openReads: [{ id: 'flat', quantity: 0 }] }, undefined),
    ).toBeUndefined();
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
    expect(
      openPositionFor([{ exchange: 'nasdaq', symbol: 'AAPL', quantity: 1 }], 'binance', 'AAPL'),
    ).toBeUndefined();
    expect(isCashEquity('nasdaq')).toBe(true);
    expect(isCashEquity('binance')).toBe(false);
  });
});
