import { describe, expect, it } from 'vitest';
import { screen } from '@testing-library/react';
import { i18n } from '@/libs/i18n';
import { renderWithTheme } from '@/test/render';
import { FillMoves } from './FillMoves';

describe('FillMoves', () => {
  it('shows a ready zero, a pending window, and a blank miss', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(
      <FillMoves
        moves={[
          { window: '1h', status: 'pending' },
          { window: '4h', status: 'ready', pct: 0, price: 100 },
          { window: '24h', status: 'missing' },
        ]}
      />,
    );
    expect(screen.getByRole('group', { name: 'After fill' })).toHaveAttribute(
      'title',
      'Venue price change after the fill. A blank window is not zero, and it is not profit.',
    );
    expect(screen.getByText('1h pending').tagName).toBe('DIV');
    expect(screen.getByText('4h 0.00%').tagName).toBe('DIV');
    expect(screen.getByText('24h —').tagName).toBe('DIV');
    expect(screen.queryByText('24h 0.00%')).not.toBeInTheDocument();
    expect(screen.queryByText('24h +0.00%')).not.toBeInTheDocument();
  });

  it('does not print zero when a missing window carries a number', async () => {
    await i18n.changeLanguage('en');
    renderWithTheme(
      <FillMoves
        moves={[
          { window: '1h', status: 'missing', pct: 0, price: 0 },
          { window: '4h', status: 'ready', pct: 1.5, price: 101.5 },
        ]}
      />,
    );
    expect(screen.getByText('1h —')).toBeInTheDocument();
    expect(screen.getByText('4h +1.50%')).toBeInTheDocument();
    expect(screen.queryByText('1h 0.00%')).not.toBeInTheDocument();
    expect(screen.queryByText('1h +0.00%')).not.toBeInTheDocument();
  });

  it('uses the Turkish labels', async () => {
    await i18n.changeLanguage('tr');
    renderWithTheme(
      <FillMoves
        moves={[
          { window: '1h', status: 'pending' },
          { window: '24h', status: 'missing' },
        ]}
      />,
    );
    expect(screen.getByRole('group', { name: 'İşlemden sonra' })).toBeInTheDocument();
    expect(screen.getByText('1h bekliyor')).toBeInTheDocument();
    expect(screen.getByText('24h —')).toBeInTheDocument();
    await i18n.changeLanguage('en');
  });
});
