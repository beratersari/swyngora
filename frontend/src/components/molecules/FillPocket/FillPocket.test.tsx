import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { renderWithTheme } from '@/test/render';
import { formatPrice } from '@/libs/utils/formatPrice';
import { FillPocket } from './FillPocket';
import { fillPocketPriceLines } from './helpers';

const pocket = {
  open: true,
  fillPrice: 100,
  fillKind: 'spot' as const,
  pocketSide: 'long' as const,
  pocketPrice: 99,
  gapPct: 1,
  pocketFromFill: 'below' as const,
  marginLiqPrice: 80.5,
  marginGapPct: 10.6,
  marginFromPocket: 'below' as const,
};

describe('FillPocket', () => {
  it('states the fill, the pocket, and the paper liquidation', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(<FillPocket mark={pocket} />);
    const group = screen.getByRole('group', { name: 'Fill and pocket' });
    expect(group).toHaveTextContent('Paper fill at 100. The nearer long pocket is 1% below it, at 99.');
    expect(group).toHaveTextContent(
      `Paper liquidation is 10.6% below that pocket, at ${formatPrice(80.5)}.`,
    );
    const lines = fillPocketPriceLines(pocket, i18n.getFixedT('en', 'detail'));
    expect(lines.map((line) => line.title)).toEqual([
      'Paper fill',
      'Long pocket · 1% below',
      'Paper liquidation',
    ]);
    expect(lines.map((line) => line.price)).toEqual([100, 99, 80.5]);
  });

  it('leaves the pocket unmarked and names the venue', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(
      <FillPocket
        mark={{
          open: true,
          fillPrice: 100,
          marginLiqPrice: 80.5,
          missing: ['okx'],
        }}
      />,
    );
    expect(screen.getByTestId('fill-pocket')).toHaveTextContent(
      'Paper fill at 100. The estimated pocket stays unmarked until the map includes OKX.',
    );
    expect(screen.getByTestId('fill-pocket')).toHaveTextContent(
      `Paper liquidation at ${formatPrice(80.5)}. The pocket stays unmarked until the map includes OKX.`,
    );
    const lines = fillPocketPriceLines(
      { open: true, fillPrice: 100, marginLiqPrice: 80.5, missing: ['okx'], pocketPrice: 99 },
      i18n.getFixedT('en', 'detail'),
    );
    expect(lines.map((line) => line.id)).toEqual(['paper-fill', 'paper-liq']);
  });

  it('draws nothing when the book is flat', () => {
    renderWithTheme(<FillPocket mark={{ open: false, symbol: 'BTCUSDT' }} />);
    expect(screen.queryByTestId('fill-pocket')).not.toBeInTheDocument();
  });

  it('uses the Turkish labels', async () => {
    await i18n.changeLanguage('tr');
    renderWithTheme(<FillPocket mark={pocket} />);
    expect(screen.getByRole('group', { name: 'İşlem ve cep' })).toHaveTextContent(
      'Kağıt işlem 100. Daha yakın long cep onun 1% altında, 99.',
    );
    await i18n.changeLanguage('en');
  });
});
