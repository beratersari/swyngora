import type { ForensicsVenue, ForensicsVenueTiming } from '@/libs/types/forensics.types';

export type ForensicsTapeProps = {
  venue?: ForensicsVenue;
  venueTiming?: ForensicsVenueTiming | null;
  isLoading?: boolean;
};
