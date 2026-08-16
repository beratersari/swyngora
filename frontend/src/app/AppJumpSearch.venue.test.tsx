import { describe, expect, it, vi } from 'vitest';
import { Route, Routes, useLocation } from 'react-router-dom';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { AppJumpSearch } from './AppJumpSearch';

vi.mock('@/components/molecules/SymbolSuggest', () => ({
  SymbolSuggest: ({
    exchange,
    onPick,
  }: {
    exchange: string;
    onPick?: (symbol: string) => void;
  }) => (
    <button
      type="button"
      data-testid="jump-pick"
      data-exchange={exchange}
      onClick={() => onPick?.('BTC-USD')}
    >
      pick
    </button>
  ),
}));

function LocationProbe() {
  const loc = useLocation();
  return <div data-testid="loc">{loc.pathname + loc.search}</div>;
}

describe('AppJumpSearch venue (finding 6)', () => {
  it('follows ?exchange= on the markets list (not hardcoded binance)', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <>
        <AppJumpSearch />
        <LocationProbe />
        <Routes>
          <Route path="/markets" element={<div>list</div>} />
          <Route path="/markets/:exchange/:symbol" element={<div>detail</div>} />
        </Routes>
      </>,
      { routerEntries: ['/markets?exchange=coinbase'] },
    );

    expect(screen.getByTestId('jump-pick')).toHaveAttribute('data-exchange', 'coinbase');
    await user.click(screen.getByTestId('jump-pick'));
    expect(screen.getByTestId('loc').textContent).toBe('/markets/coinbase/BTC-USD');
  });

  it('still reads the venue from /markets/:exchange/:symbol', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <>
        <AppJumpSearch />
        <LocationProbe />
        <Routes>
          <Route path="/markets/:exchange/:symbol" element={<div>detail</div>} />
        </Routes>
      </>,
      { routerEntries: ['/markets/bybit/ETHUSDT'] },
    );

    expect(screen.getByTestId('jump-pick')).toHaveAttribute('data-exchange', 'bybit');
    await user.click(screen.getByTestId('jump-pick'));
    expect(screen.getByTestId('loc').textContent).toBe('/markets/bybit/BTC-USD');
  });
});
