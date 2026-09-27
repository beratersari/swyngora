import { Navigate, Route, Routes } from 'react-router-dom';
import { AiChatPage } from '@/components/pages/AiChatPage';
import { AlertsPage } from '@/components/pages/AlertsPage';
import { BasisPage } from '@/components/pages/BasisPage';
import { ArbPage } from '@/components/pages/ArbPage';
import { CoinDetailPage } from '@/components/pages/CoinDetailPage';
import { ComparePage } from '@/components/pages/ComparePage';
import { ForensicsPage } from '@/components/pages/ForensicsPage';
import { HeatmapPage } from '@/components/pages/HeatmapPage';
import { LiquidationsPage } from '@/components/pages/LiquidationsPage';
import { OptionsPage } from '@/components/pages/OptionsPage';
import { MarketsPage } from '@/components/pages/MarketsPage';
import { PortfolioPage } from '@/components/pages/PortfolioPage';
import { PumpsPage } from '@/components/pages/PumpsPage';
import { SurgesPage } from '@/components/pages/SurgesPage';
import { OrderCountPage } from '@/components/pages/OrderCountPage';
import { PrintSizePage } from '@/components/pages/PrintSizePage';
import { ImpactPage } from '@/components/pages/ImpactPage';
import { WallsPage } from '@/components/pages/WallsPage';
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
      <Route path="/options" element={<OptionsPage />} />
      <Route path="/basis" element={<BasisPage />} />
      <Route path="/portfolio" element={<PortfolioPage />} />
      <Route path="/signals" element={<SignalsPage />} />
      <Route path="/pumps" element={<PumpsPage />} />
      <Route path="/surges" element={<SurgesPage />} />
      <Route path="/order-count" element={<OrderCountPage />} />
      <Route path="/print-size" element={<PrintSizePage />} />
      <Route path="/impact" element={<ImpactPage />} />
      <Route path="/walls" element={<WallsPage />} />
      <Route path="/alerts" element={<AlertsPage />} />
      <Route path="/arb" element={<ArbPage />} />
      <Route path="/compare" element={<ComparePage />} />
      <Route path="/forensics" element={<ForensicsPage />} />
      <Route path="/heatmap" element={<HeatmapPage />} />
      <Route path="/settings" element={<SettingsPage />} />
      <Route path="/ai" element={<AiChatPage />} />
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
