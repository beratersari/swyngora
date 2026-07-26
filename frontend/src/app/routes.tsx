import { Navigate, Route, Routes } from 'react-router-dom';
import { CoinDetailPage } from '@/components/pages/CoinDetailPage';
import { MarketsPage } from '@/components/pages/MarketsPage';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/markets" replace />} />
      <Route path="/markets" element={<MarketsPage />} />
      <Route path="/markets/:exchange/:symbol" element={<CoinDetailPage />} />
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
