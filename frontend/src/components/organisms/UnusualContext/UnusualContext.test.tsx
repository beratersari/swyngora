import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import tr from '@/libs/i18n/locales/tr/detail.json';
import { renderWithTheme } from '@/test/render';
import { UnusualContext } from './UnusualContext';
import type { UnusualReportView } from './UnusualContext.types';

const report: UnusualReportView = {
  symbol: 'BTCUSDT',
  exchange: 'all',
  note: 'DO NOT SHOW NOTE',
  venues: [
    {
      exchange: 'binance',
      readings: [
        {
          metric: 'funding',
          band: 'extreme_high',
          label: 'settled',
          current: 0.0001,
          percentile: 99,
          samples: 40,
          minSamples: 20,
          spanHours: 29 * 24,
          unit: 'rate',
          summary: 'DO NOT SHOW',
        },
        {
          metric: 'print_size',
          band: 'unknown',
          reason: 'no_stored_history',
          current: 0,
          summary: 'DO NOT SHOW',
        },
      ],
    },
    {
      exchange: 'okx',
      readings: [
        {
          metric: 'spread',
          band: 'unknown',
          reason: 'no_book',
          current: 0,
          summary: 'DO NOT SHOW',
        },
      ],
    },
  ],
};

describe('UnusualContext', () => {
  it('builds the sentence from localized copy and hides the English API summary', () => {
    renderWithTheme(<UnusualContext report={report} />);
    expect(screen.getByText('Versus its own history')).toBeTruthy();
    expect(screen.getByTestId('unusual-binance-funding').textContent).toContain(
      'Settled funding is 0.0100%, extremely high versus 40 stored settlements over 29 days (percentile 99).',
    );
    expect(screen.getByTestId('unusual-binance-print_size').textContent).toContain(
      'previous 15 minutes',
    );
    expect(screen.getByTestId('unusual-okx-spread').textContent).toContain('no spot book');
    expect(screen.queryByText(/DO NOT SHOW/)).toBeNull();
    expect(screen.getByText('Binance')).toBeTruthy();
    expect(screen.getByText('OKX')).toBeTruthy();
  });

  it('says how many samples are still needed', () => {
    renderWithTheme(
      <UnusualContext
        report={{
          venues: [
            {
              exchange: 'bybit',
              readings: [
                {
                  metric: 'cvd',
                  band: 'unknown',
                  reason: 'few_samples',
                  samples: 3,
                  minSamples: 20,
                  spanHours: 48,
                  summary: 'DO NOT SHOW',
                },
              ],
            },
          ],
        }}
      />,
    );
    expect(screen.getByTestId('unusual-bybit-cvd').textContent).toContain(
      'CVD has 3 stored samples over 2 days. At least 20 are needed',
    );
  });

  it('stays empty until a report, a load, or an error exists', () => {
    const { container } = renderWithTheme(<UnusualContext />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows a checking line while the first read is in flight', () => {
    renderWithTheme(<UnusualContext isLoading />);
    expect(screen.getByText('Checking stored history…')).toBeTruthy();
  });

  it('keeps a Turkish sentence for the same reading', () => {
    expect(tr.tape.unusual.title).toBe('Kendi geçmişine göre');
    expect(tr.tape.unusual.band.extreme_high).toBe('aşırı yüksek');
    expect(tr.tape.unusual.kind.settled).toBe('Kesinleşmiş fonlama');
    expect(tr.tape.unusual.printSize).toContain('15 dakika');
    expect(tr.tape.unusual.line.funding).toContain('{{band}}');
  });
});
