import { Navigate, Route, Routes } from 'react-router-dom';
import { AiChatPage } from '@/components/pages/AiChatPage';
import { AlertsPage } from '@/components/pages/AlertsPage';
import { CoinDetailPage } from '@/components/pages/CoinDetailPage';
import { ComparePage } from '@/components/pages/ComparePage';
import { HeatmapPage } from '@/components/pages/HeatmapPage';
import { LiquidationsPage } from '@/components/pages/LiquidationsPage';
import { MarketsPage } from '@/components/pages/MarketsPage';
import { PortfolioPage } from '@/components/pages/PortfolioPage';
import { PumpsPage } from '@/components/pages/PumpsPage';
import { SignalsPage } from '@/components/pages/SignalsPage';
import { SettingsPage } from '@/components/pages/SettingsPage';
import { WatchlistPage } from '@/components/pages/WatchlistPage';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/markets" replace />} />
      <Route path="/markets" element={<MarketsPage />} />
      <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      <Route path="/watchlist" element={<WatchlistPage />} />
      <Route path="/liquidations" element={<LiquidationsPage />} />
      <Route path="/portfolio" element={<PortfolioPage />} />
      <Route path="/signals" element={<SignalsPage />} />
      <Route path="/pumps" element={<PumpsPage />} />
      <Route path="/alerts" element={<AlertsPage />} />
      <Route path="/compare" element={<ComparePage />} />
      <Route path="/heatmap" element={<HeatmapPage />} />
      <Route path="/settings" element={<SettingsPage />} />
      <Route path="/ai" element={<AiChatPage />} />
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
