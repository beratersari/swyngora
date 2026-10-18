import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { renderWithTheme } from '@/test/render';
import { formatPrice } from '@/libs/utils/formatPrice';
import { FillCostCheck } from './FillCostCheck';
import { signedQuote } from './helpers';

const paidMore = {
  expectedFee: 10,
  actualFee: 10.4,
  feeDelta: 0.4,
  expectedSlippage: 15,
  actualSlippage: 18.2,
  slippageDelta: 3.2,
  quotedQuantity: 1,
  filledQuantity: 1,
};

describe('FillCostCheck', () => {
  it('shows the quoted fee and slippage beside what was paid', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(<FillCostCheck check={paidMore} />);
    const view = screen.getByTestId('fill-cost-check');
    expect(view).toHaveTextContent(`Fee paid ${formatPrice(10.4)}`);
    expect(view).toHaveTextContent(`estimate ${formatPrice(10)}`);
    expect(view).toHaveTextContent(signedQuote(0.4));
    expect(view).toHaveTextContent(`Slippage ${formatPrice(18.2)}`);
    expect(view).toHaveTextContent(`estimate ${formatPrice(15)}`);
    expect(view).toHaveTextContent(signedQuote(3.2));
    expect(screen.queryByText(/Filled/)).not.toBeInTheDocument();
  });

  it('names a fill that stopped short of the quoted size', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(
      <FillCostCheck check={{ ...paidMore, filledQuantity: 0.4, slippageDelta: -1 }} />,
    );
    expect(
      screen.getByText(`Filled ${formatPrice(0.4)} of the quoted ${formatPrice(1)}`),
    ).toBeInTheDocument();
    expect(screen.getByTestId('fill-cost-check')).toHaveTextContent(signedQuote(-1));
  });

  it('uses the Turkish labels', async () => {
    await i18n.changeLanguage('tr');
    renderWithTheme(<FillCostCheck check={paidMore} />);
    expect(screen.getByTestId('fill-cost-check')).toHaveTextContent('Ödenen komisyon');
    expect(screen.getByTestId('fill-cost-check')).toHaveTextContent('Kayma');
    await i18n.changeLanguage('en');
  });
});
