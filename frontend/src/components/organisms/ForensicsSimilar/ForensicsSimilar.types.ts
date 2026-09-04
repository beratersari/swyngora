import type { ForensicsSimilar } from '@/libs/types/forensics.types';

export type ForensicsSimilarProps = {
  similar?: ForensicsSimilar | null;
  similarError?: string;
  isLoading?: boolean;
};
