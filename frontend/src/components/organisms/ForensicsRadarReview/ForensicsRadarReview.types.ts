import type { ForensicsRadarReview } from '@/libs/types/forensics.types';

export type ForensicsRadarReviewProps = {
  review?: ForensicsRadarReview;
  isLoading?: boolean;
  onOpenCase?: (symbol: string, foundAt: string) => void;
};
