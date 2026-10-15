import { describe, expect, it } from 'vitest';
import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { DerivativesTimeGrid } from './DerivativesTimeGrid';
import type { GridMetric } from './constants';

const grid = {
  columns: ['2026-10-10T14:00:00Z', '2026-10-10T15:00:00Z'],
  rows: [
    {
      symbol: 'BTCUSDT',
      base: 'BTC',
      cells: [
        {
          time: '2026-10-10T14:00:00Z',
          definite: false,
          missing: ['hyperliquid'],
        },
        {
          time: '2026-10-10T15:00:00Z',
          definite: true,
          value: 0.0001,
          text: '+0.01%',
          venues: [
            { exchange: 'binance', definite: true, value: 0.0001, text: '+0.0100%' },
            { exchange: 'okx', definite: false },
          ],
        },
      ],
    },
  ],
};

function renderGrid(metric: GridMetric = 'funding') {
  return renderWithProviders(
    <DerivativesTimeGrid exchange="all" metric={metric} range="24h" grid={grid} loading={false} />,
  );
}

describe('DerivativesTimeGrid', () => {
  it('opens a missing exchange when the cell has no venue rows', async () => {
    const user = userEvent.setup();
    renderGrid();
    await user.click(screen.getByRole('button', { name: /BTC at 14:00/ }));
    const dialog = await screen.findByRole('dialog', { name: /BTC · 14:00/ });
    expect(within(dialog).getByText('Hyperliquid')).toBeInTheDocument();
    expect(within(dialog).getByText('Missing. Not counted as zero.')).toBeInTheDocument();
    expect(within(dialog).queryByText('Combined')).not.toBeInTheDocument();
    expect(within(dialog).queryByText('+0.0100%')).not.toBeInTheDocument();
  });

  it('closes the detail when the series changes', async () => {
    const user = userEvent.setup();
    const view = renderGrid('funding');
    await user.click(screen.getByRole('button', { name: /BTC at 15:00/ }));
    const dialog = await screen.findByRole('dialog', { name: /BTC · 15:00/ });
    expect(within(dialog).getByText('Binance')).toBeInTheDocument();
    expect(within(dialog).getByText('+0.0100%')).toBeInTheDocument();
    expect(within(dialog).getByText('OKX')).toBeInTheDocument();
    expect(within(dialog).getByText('Missing. Not counted as zero.')).toBeInTheDocument();
    expect(within(dialog).getByText('Combined')).toBeInTheDocument();

    view.rerender(
      <DerivativesTimeGrid exchange="all" metric="oi" range="24h" grid={grid} loading={false} />,
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});
