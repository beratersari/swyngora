import type { PortfolioStressReport } from '@/libs/api';

export type StressMove = { asset: string; movePct: number };

export type StressMode = 'moves' | 'period';

export type StressPreviewRequest = { moves: StressMove[] } | { from: string; to: string };

export type PortfolioStressPanelProps = {
  disabled?: boolean;
  isLoading?: boolean;
  report?: PortfolioStressReport | null;
  errorMessage?: string | null;
  onPreview: (request: StressPreviewRequest) => void;
};
