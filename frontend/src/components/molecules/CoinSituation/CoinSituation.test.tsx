import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { formatDateTime } from '@/libs/utils/formatMarket';
import { renderWithProviders } from '@/test/render';
import { CoinSituation } from './CoinSituation';

describe('CoinSituation', () => {
  it('names a missing venue and keeps that clause blank of a number', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'price_up_cvd_down' },
          openInterestMissing: ['bybit', 'okx'],
          liquidation: { side: 'long', distancePct: 1.2 },
        }}
      />,
    );
    expect(screen.getByRole('region', { name: 'Situation' })).toBeInTheDocument();
    expect(screen.getByText('Price is rising while futures CVD is falling.')).toBeInTheDocument();
    expect(
      screen.getByText('Open interest stays blank until Bybit and OKX have the last hour.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Long liquidations below price are closer (1.2%).')).toBeInTheDocument();
    expect(screen.queryByText(/building/)).not.toBeInTheDocument();
  });

  it('states the leverage model and the measured prints as two facts', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          liquidation: { side: 'long', distancePct: 1.2 },
          prints: { side: 'short', longNotional: 3_200, shortNotional: 2_500_000, window: '24h' },
        }}
      />,
    );
    expect(screen.getByText('Long liquidations below price are closer (1.2%).')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Measured prints over the last 24h are mostly short liquidations (3.20K long, 2.50M short).',
      ),
    ).toBeInTheDocument();
  });

  it('names the venues that leave measured prints blank', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          printsMissing: ['okx'],
        }}
      />,
    );
    expect(
      screen.getByText('Measured liquidation prints stay blank until OKX cover the last 24h.'),
    ).toBeInTheDocument();
  });

  it('adds funding, the crowd gap, and whale versus retail when those clauses are set', () => {
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          funding: { payer: 'long', ratePct: 0.01 },
          crowd: { lean: 'top_more_long', gapPct: 7 },
          size: { kind: 'whale_buy_retail_sell' },
          sizeMissing: [],
        }}
      />,
    );
    expect(screen.getByText('Longs pay the combined funding (0.01%).')).toBeInTheDocument();
    expect(screen.getByText('Top traders are more long than the crowd (7 pp).')).toBeInTheDocument();
    expect(screen.getByText('Large prints are buying while smaller prints are selling.')).toBeInTheDocument();
  });

  it('says how long a live clause has held and stays quiet when the clock is blank', async () => {
    await i18n.changeLanguage('en');
    const since = '2026-10-12T11:30:00Z';
    const time = formatDateTime(since, i18n.language);
    renderWithProviders(
      <CoinSituation
        report={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'price_up_cvd_down', since },
          openInterestMissing: ['bybit'],
          liquidation: { side: 'long', distancePct: 1.2 },
        }}
      />,
    );
    expect(
      screen.getByText(`Price is rising while futures CVD is falling. Since ${time}.`),
    ).toBeInTheDocument();
    expect(screen.getByText('Long liquidations below price are closer (1.2%).')).toBeInTheDocument();
    expect(screen.queryByText(/closer \(1\.2%\)\. Since/)).not.toBeInTheDocument();
    expect(
      screen.getByText('Open interest stays blank until Bybit have the last hour.'),
    ).toBeInTheDocument();
    await i18n.changeLanguage('tr');
    const trTime = formatDateTime(since, i18n.language);
    expect(
      screen.getByText(`Fiyat yükselirken vadeli CVD düşüyor. ${trTime} tarihinden beri.`),
    ).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });
});
