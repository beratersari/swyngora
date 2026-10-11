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
          windows: [
            { window: '1h', changeValue: '+300000000', changeValuePct: '+14.3', flow: 'opened', complete: true },
            { window: '4h', changeValue: '+500000000', changeValuePct: '+25', flow: 'opened', complete: true },
            { window: '24h', changeValue: '-100000000', changeValuePct: '-4.2', flow: 'closed', complete: true },
          ],
          venues: [
            {
              exchange: 'binance',
              current: { value: '800000000' },
              windows: [{ window: '1h', changeValue: '+200000000', changeValuePct: '+12', flow: 'opened' }],
            },
          ],
          funding: { current: { ratePct: '0.01', payer: 'long' } },
        }}
        liquidations={{
          windows: [{ window: '1h', totalNotional: '25000000', complete: true }],
        }}
        cvd={{
          combined: {
            lastCvd: '12.5',
            summary: 'buy flow',
            anchors: [
              { anchor: 'day', cvd: '+1200000', complete: true, summary: 'Since the day open (00:00 UTC), CVD +1.20M.' },
              { anchor: '4h', cvd: '-400000', complete: false, summary: 'Since the 4h open (08:00 UTC), CVD -400.00K.' },
            ],
          },
        }}
        snapshot={{
          summary: 'Volume building ahead of price',
          combined: { longPct: '55.2', windows: [{ window: '1h', taker: { dominant: 'buy', buyShare: '0.58' } }] },
        }}
        participation={{
          combined: {
            windows: [
              { window: '15m', volume: '2000000000', oiDelta: '+300000000', ratioPct: '+15', lean: 'entering' },
              { window: '1h', ratioPct: '8' },
              { window: '4h', ratioPct: '12' },
            ],
            summary: '15m: 2B volume, OI +300M (15% of volume) is new positions.',
          },
        }}
        symbol="BTCUSDT"
        botTracker={{
          delta: 6,
          buyCount: 6,
          sellCount: 0,
          summary: 'Repeated size: buy 57.89K × 6 in 30s.',
          clusters: [{ side: 'buy', notional: '57889', count: 6, exchange: 'binance' }],
        }}
        orderCount={{
          summary: '5m: 80 buys, 20 sells, ratio +0.60 (more market buys).',
          combined: {
            windows: [
              { window: '1m', ratio: '+0.40', buyCount: 7, sellCount: 3 },
              { window: '5m', ratio: '+0.60', buyCount: 80, sellCount: 20 },
              { window: '15m', ratio: '-0.10', buyCount: 45, sellCount: 55 },
            ],
          },
        }}
        perpLiquidity={{
          bid: '2400000000',
          ask: '1800000000',
          summary: 'BTC 48% of that book.',
          coins: [{ symbol: 'BTCUSDT', base: 'BTC', sharePct: '48', bidSharePct: '60' }],
          windows: [
            {
              window: '1h',
              complete: true,
              pastBidPct: '55',
              bidPct: '65',
              changePct: '+10',
              pastTotal: '400000000',
              total: '500000000',
              totalChangePct: '+25',
            },
          ],
        }}
        volumeShare={{
          windows: [
            {
              window: '1h',
              total: '100',
              venues: [
                { exchange: 'binance', sharePct: '55', kind: 'gained' },
                { exchange: 'bybit', sharePct: '30', kind: 'lost' },
                { exchange: 'okx', sharePct: '15', kind: 'stable' },
              ],
              gained: ['Binance'],
              lost: ['Bybit'],
              summary: '1h: Binance 55%, Bybit 30%, and OKX 15%. Binance gained share; Bybit lost share.',
            },
          ],
        }}
      />,
    );
    expect(screen.getByText('Futures tape')).toBeInTheDocument();
    expect(screen.getByText('1.00B')).toBeInTheDocument();
    expect(screen.getByText(/\+300\.00M \(\+14\.3%\) opened/i)).toBeInTheDocument();
    expect(screen.getAllByText(/binance/i).length).toBeGreaterThan(0);
    expect(screen.getByText('0.0100%')).toBeInTheDocument();
    expect(screen.getByText('25.00M')).toBeInTheDocument();
    expect(screen.getByText('buy flow')).toBeInTheDocument();
    expect(screen.getByText('Repeated size')).toBeInTheDocument();
    expect(screen.getByText('Repeated buy size')).toBeInTheDocument();
    expect(screen.getByText('57.89K × 6')).toBeInTheDocument();
    expect(screen.getByText('Repeated buy prints')).toBeInTheDocument();
    expect(screen.getByText(/only the repeated clips/)).toBeInTheDocument();
    expect(screen.getByText('Order count')).toBeInTheDocument();
    expect(screen.getByText('+0.60')).toBeInTheDocument();
    expect(screen.getByText('5m buy prints')).toBeInTheDocument();
    expect(screen.getByText(/Repeated size: buy 57.89K/)).toBeInTheDocument();
    expect(screen.getByText('Since day open')).toBeInTheDocument();
    expect(screen.getByText('1.20M')).toBeInTheDocument();
    expect(screen.getByText(/-400\.00K \(Partial\)/)).toBeInTheDocument();
    expect(screen.getByText(/Since the day open/)).toBeInTheDocument();
    expect(screen.getByText('Volume building ahead of price')).toBeInTheDocument();
    expect(screen.getByText('+15%')).toBeInTheDocument();
    expect(screen.getByText('55%')).toBeInTheDocument();
    expect(screen.getByText('48%')).toBeInTheDocument();
    expect(screen.getByText('55% → 65% (+10)')).toBeInTheDocument();
    expect(screen.getByText('400.00M → 500.00M (+25%)')).toBeInTheDocument();
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

  it('shows average buy vs sell print size', () => {
    renderWithTheme(
      <TapePanel
        printSize={{
          combined: {
            summary: 'Buy prints averaged 180000 USDT vs 20000 before.',
            windows: [
              {
                window: '15m',
                buy: { avg: '180000', priorAvg: '20000', kind: 'jumped' },
                sell: { avg: '25000', priorAvg: '25000', kind: 'stable' },
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText('180.00K')).toBeInTheDocument();
    expect(screen.getByText('20.00K')).toBeInTheDocument();
    expect(screen.getByText(/few large|180000/i)).toBeInTheDocument();
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

  it('shows crowd vs top gap and whether it is widening', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          longShort: {
            venues: [
              {
                exchange: 'binance',
                current: { longPct: '55.00', shortPct: '45.00', bias: 'long' },
                topPositions: { longPct: '65.00', shortPct: '35.00', bias: 'long' },
                gap: {
                  versus: 'top_positions',
                  crowdLongPct: '55',
                  topLongPct: '65',
                  gapPct: '+10',
                  lean: 'top_more_long',
                  headline: 'Top traders are 10pp more long than other accounts.',
                },
                gapWindows: [
                  { window: '1h', kind: 'widened', changePct: '+6', complete: true },
                  { window: '24h', kind: 'stable', complete: false },
                ],
              },
            ],
          },
        }}
        snapshot={{ combined: { longPct: '55.00', windows: [] } }}
      />,
    );
    expect(screen.getByText('+10 pp')).toBeInTheDocument();
    expect(screen.getByText(/10pp more long/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Widening/i).length).toBeGreaterThan(0);
  });

  it('shows OKX crowd and top size beside the Binance read', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          longShort: {
            venues: [
              {
                exchange: 'binance',
                current: { longPct: '55.00', shortPct: '45.00', bias: 'long' },
              },
              {
                exchange: 'okx',
                current: { longPct: '52.00', shortPct: '48.00', bias: 'long' },
                topPositions: { longPct: '71.00', shortPct: '29.00', bias: 'long' },
                gap: { gapPct: '+19', lean: 'top_more_long' },
              },
            ],
          },
        }}
        snapshot={{ combined: { longPct: '55.00', windows: [] } }}
      />,
    );
    expect(screen.getByText(/okx 52.00%/i)).toBeInTheDocument();
    expect(screen.getByText(/Top size long % 71.00%/i)).toBeInTheDocument();
    expect(screen.getByText(/Crowd vs top \+19 pp/i)).toBeInTheDocument();
  });

  it('shows top vs price divergence and when it started', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          longShort: {
            venues: [
              {
                exchange: 'binance',
                current: { longPct: '55', shortPct: '45', bias: 'long' },
                topPositions: { longPct: '65', shortPct: '35', bias: 'long' },
                divergence: {
                  kind: 'top_long_price_down',
                  duration: '30m',
                  since: '2026-09-20T15:30:00Z',
                  priceChangePct: '-1.5',
                  topChangePct: '+3',
                  headline: 'Top traders got more long while price dropped 1.5% over 30m.',
                  detail: 'Started 15:30 UTC.',
                },
              },
            ],
          },
        }}
        snapshot={{ combined: { longPct: '55', windows: [] } }}
      />,
    );
    expect(screen.getByText('30m')).toBeInTheDocument();
    expect(screen.getByText(/dropped 1\.5%/i)).toBeInTheDocument();
    expect(screen.getByText(/15:30 UTC/i)).toBeInTheDocument();
  });

  it('shows combined net pressure and 1h change', () => {
    renderWithTheme(
      <TapePanel
        pressure={{
          current: {
            value: '+0.40',
            lean: 'buy',
            agreement: 'opposite',
            headline:
              'The order book looks bid-heavy while CVD is selling, but bids are absorbing that flow. Net pressure +0.40.',
          },
          change1h: '+0.22',
          windows: [{ window: '1h', change: '+0.22', kind: 'more_buy', complete: true }],
        }}
      />,
    );
    expect(screen.getByText('+0.40')).toBeInTheDocument();
    expect(screen.getByText('+0.22')).toBeInTheDocument();
    expect(screen.getByText(/bids are absorbing/i)).toBeInTheDocument();
  });

  it('blanks a combined number when a venue is missing', () => {
    renderWithTheme(
      <TapePanel
        openInterest={{
          missing: ['bybit'],
          current: { value: '1000000000' },
          windows: [
            {
              window: '1h',
              changeValue: '+300000000',
              changeValuePct: '+14.3',
              flow: 'opened',
              complete: false,
              missing: ['bybit'],
            },
            {
              window: '4h',
              changeValue: '+500000000',
              changeValuePct: '+25',
              flow: 'opened',
              complete: true,
            },
          ],
          venues: [
            {
              exchange: 'binance',
              current: { value: '800000000' },
              windows: [{ window: '1h', changeValue: '+200000000', changeValuePct: '+12', flow: 'opened' }],
            },
          ],
          funding: {
            current: { ratePct: '0.05', payer: 'long' },
            pressure: {
              ratePct: '0.007778',
              payer: 'long',
              missing: ['okx'],
              summary: 'Combined funding pressure is withheld until okx is in.',
            },
            pressureWindows: [
              { window: '1h', changePct: '+0.001', complete: true },
              { window: '24h', changePct: '-0.002', complete: true },
            ],
          },
        }}
        cvd={{
          venues: [{ exchange: 'bybit', lastCvd: '99' }],
          combined: {
            complete: false,
            lastCvd: '12.5',
            missing: ['binance'],
            summary: 'Combined CVD leaves out minutes binance missed.',
            anchors: [
              {
                anchor: 'day',
                cvd: '+1200000',
                complete: false,
                summary: 'Since the day open (00:00 UTC), CVD +1.20M.',
              },
            ],
          },
        }}
        participation={{
          venues: [{ windows: [{ window: '15m', ratioPct: '+40', complete: true }] }],
          combined: {
            windows: [{ window: '15m', volume: '2000000000', oiDelta: '+300000000', ratioPct: '+15', complete: false }],
          },
        }}
      />,
    );
    expect(screen.queryByText(/1\.00B/)).not.toBeInTheDocument();
    expect(screen.queryByText(/\+300\.00M/)).not.toBeInTheDocument();
    expect(screen.queryByText('0.0078%')).not.toBeInTheDocument();
    expect(screen.queryByText('0.0500%')).not.toBeInTheDocument();
    expect(screen.queryByText('+0.0010%')).not.toBeInTheDocument();
    expect(screen.queryByText('-0.0020%')).not.toBeInTheDocument();
    expect(screen.queryByText('Longs pay')).not.toBeInTheDocument();
    expect(screen.queryByText(/12\.5/)).not.toBeInTheDocument();
    expect(screen.queryByText('99')).not.toBeInTheDocument();
    expect(screen.queryByText(/Since the day open/)).not.toBeInTheDocument();
    expect(screen.queryByText('1.20M')).not.toBeInTheDocument();
    expect(screen.queryByText(/\(Partial\)/)).not.toBeInTheDocument();
    expect(screen.queryByText('+15%')).not.toBeInTheDocument();
    expect(screen.queryByText('+40%')).not.toBeInTheDocument();
    expect(screen.getByText(/800\.00M/)).toBeInTheDocument();
    expect(screen.getByText(/\+200\.00M/)).toBeInTheDocument();
    expect(screen.getByText(/\+500\.00M/)).toBeInTheDocument();
    expect(screen.getByText(/until bybit is in/i)).toBeInTheDocument();
    expect(screen.getByText(/until okx is in/i)).toBeInTheDocument();
    expect(screen.getByText(/until binance is in/i)).toBeInTheDocument();
    expect(screen.getByText(/leaves out minutes binance/i)).toBeInTheDocument();
  });

  it('explains a missing perp book', () => {
    renderWithTheme(<TapePanel openInterestError="no perpetual on this venue" />);
    expect(screen.getAllByText('no perpetual on this venue').length).toBeGreaterThan(0);
  });
});
