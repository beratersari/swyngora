import type { ForensicsTimeline } from '@/libs/types/forensics.types';

export type ForensicsTimelineProps = {
  timeline?: ForensicsTimeline | null;
  isLoading?: boolean;
};
