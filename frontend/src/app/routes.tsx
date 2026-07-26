import { Navigate, Route, Routes } from 'react-router-dom';
import { MarketsPage } from '@/components/pages/MarketsPage';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/markets" replace />} />
      <Route path="/markets" element={<MarketsPage />} />
      <Route path="*" element={<Navigate to="/markets" replace />} />
    </Routes>
  );
}
