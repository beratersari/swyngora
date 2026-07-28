import { Navigate, Route, Routes } from 'react-router-dom';
import { CoinDetailPage } from '@/components/pages/CoinDetailPage';
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
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
