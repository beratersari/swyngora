import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '@/test/render';
import { AppRoutes } from './routes';

vi.mock('@/components/pages/MarketsPage', () => ({
  MarketsPage: () => <div data-testid="markets-page">Markets</div>,
}));
vi.mock('@/components/pages/CoinDetailPage', () => ({
  CoinDetailPage: () => <div data-testid="detail-page">Detail</div>,
}));
vi.mock('@/components/pages/WatchlistPage', () => ({
  WatchlistPage: () => <div data-testid="watchlist-page">Watchlist</div>,
}));
vi.mock('@/components/pages/PumpsPage', () => ({
  PumpsPage: () => <div data-testid="pumps-page">Pumps</div>,
}));
vi.mock('@/components/pages/AiChatPage', () => ({
  AiChatPage: () => <div data-testid="ai-page">AI</div>,
}));
vi.mock('@/components/pages/HeatmapPage', () => ({
  HeatmapPage: () => <div data-testid="heatmap-page">Heatmap</div>,
}));
vi.mock('@/components/pages/LiquidationsPage', () => ({
  LiquidationsPage: () => <div data-testid="liquidations-page">Liquidations</div>,
}));
vi.mock('@/components/pages/SettingsPage', () => ({
  SettingsPage: () => <div data-testid="settings-page">Settings</div>,
}));

describe('AppRoutes', () => {
  it('redirects / to /markets', () => {
    renderWithProviders(<AppRoutes />, { routerEntries: ['/'] });
    expect(screen.getByTestId('markets-page')).toBeInTheDocument();
  });

  it('renders markets page', () => {
    renderWithProviders(<AppRoutes />, { routerEntries: ['/markets'] });
    expect(screen.getByTestId('markets-page')).toBeInTheDocument();
  });

  it('renders coin detail page', () => {
    renderWithProviders(<AppRoutes />, {
      routerEntries: ['/markets/binance/BTCUSDT'],
    });
    expect(screen.getByTestId('detail-page')).toBeInTheDocument();
  });

  it('renders watchlist, pumps, and ai pages', () => {
    renderWithProviders(<AppRoutes />, { routerEntries: ['/watchlist'] });
    expect(screen.getByTestId('watchlist-page')).toBeInTheDocument();
    renderWithProviders(<AppRoutes />, { routerEntries: ['/pumps'] });
    expect(screen.getByTestId('pumps-page')).toBeInTheDocument();
    renderWithProviders(<AppRoutes />, { routerEntries: ['/ai'] });
    expect(screen.getByTestId('ai-page')).toBeInTheDocument();
    renderWithProviders(<AppRoutes />, { routerEntries: ['/heatmap'] });
    expect(screen.getByTestId('heatmap-page')).toBeInTheDocument();
    renderWithProviders(<AppRoutes />, { routerEntries: ['/liquidations'] });
    expect(screen.getByTestId('liquidations-page')).toBeInTheDocument();
    renderWithProviders(<AppRoutes />, { routerEntries: ['/settings'] });
    expect(screen.getByTestId('settings-page')).toBeInTheDocument();
  });

  it('redirects unknown paths to markets', () => {
    renderWithProviders(<AppRoutes />, { routerEntries: ['/nope'] });
    expect(screen.getByTestId('markets-page')).toBeInTheDocument();
  });
});
