import { Navigate, Route, Routes } from 'react-router-dom';
import { AiChatPage } from '@/components/pages/AiChatPage';
import { AlertsPage } from '@/components/pages/AlertsPage';
import { CoinDetailPage } from '@/components/pages/CoinDetailPage';
import { ComparePage } from '@/components/pages/ComparePage';
import { MarketsPage } from '@/components/pages/MarketsPage';
import { PumpsPage } from '@/components/pages/PumpsPage';
import { WatchlistPage } from '@/components/pages/WatchlistPage';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/markets" replace />} />
      <Route path="/markets" element={<MarketsPage />} />
      <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      <Route path="/watchlist" element={<WatchlistPage />} />
      <Route path="/pumps" element={<PumpsPage />} />
      <Route path="/alerts" element={<AlertsPage />} />
      <Route path="/compare" element={<ComparePage />} />
      <Route path="/ai" element={<AiChatPage />} />
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
