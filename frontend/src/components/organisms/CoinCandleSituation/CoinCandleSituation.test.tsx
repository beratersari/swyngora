import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { renderWithProviders } from '@/test/render';
import { CoinCandleSituation } from './CoinCandleSituation';

describe('CoinCandleSituation', () => {
  it('reads the stored minute and leaves the estimated pocket off', async () => {
    await i18n.changeLanguage('en');
    renderWithProviders(
      <CoinCandleSituation
        timeLabel="2026-06-01 16:00"
        report={{
          symbol: 'BTCUSDT',
          priceCvd: { kind: 'agrees' },
          priceCvdMissing: [],
        }}
      />,
    );
    const card = screen.getByRole('region', { name: 'At this candle' });
    expect(card).toHaveTextContent('Stored minutes at 2026-06-01 16:00.');
    expect(card).toHaveTextContent('The estimated liquidation pocket is not part of this read.');
    expect(card).toHaveTextContent('Price and futures CVD are moving together.');
    expect(screen.queryByText(/closer/)).not.toBeInTheDocument();
    expect(screen.queryByText(/No estimated liquidation pocket/)).not.toBeInTheDocument();
  });

  it('names a venue the archive does not have', () => {
    renderWithProviders(
      <CoinCandleSituation
        timeLabel="2026-06-01 16:00"
        report={{ symbol: 'BTCUSDT', priceCvdMissing: ['okx'] }}
      />,
    );
    expect(
      screen.getByText('Price versus CVD stays blank until OKX cover the tape.'),
    ).toBeInTheDocument();
  });

  it('shows loading and a failed read', () => {
    const { rerender } = renderWithProviders(
      <CoinCandleSituation timeLabel="2026-06-01 16:00" isLoading />,
    );
    expect(screen.getByTestId('chart-candle-situation-loading')).toHaveTextContent(
      'Reading the stored minutes…',
    );
    rerender(
      <CoinCandleSituation timeLabel="2026-06-01 16:00" error="The candle read did not load." />,
    );
    expect(screen.getByTestId('chart-candle-situation-error')).toHaveTextContent(
      'The candle read did not load.',
    );
  });

  it('uses the Turkish title', async () => {
    await i18n.changeLanguage('tr');
    renderWithProviders(
      <CoinCandleSituation
        timeLabel="2026-06-01 16:00"
        report={{ symbol: 'BTCUSDT', priceCvd: { kind: 'agrees' } }}
      />,
    );
    expect(screen.getByRole('region', { name: 'Bu mumda' })).toHaveTextContent(
      'Tahmini tasfiye cebi bu okumada yok.',
    );
    expect(screen.getByText('Fiyat ile vadeli CVD aynı yönde.')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });

  it('stays hidden until a candle is chosen', () => {
    renderWithProviders(<CoinCandleSituation timeLabel={null} report={{ symbol: 'BTCUSDT' }} />);
    expect(screen.queryByTestId('chart-candle-situation')).not.toBeInTheDocument();
  });
});
