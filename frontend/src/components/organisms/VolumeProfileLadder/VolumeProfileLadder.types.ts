import type { VolumeProfileReport } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type VolumeProfileLadderProps = WithLoadingProps & {
  report?: VolumeProfileReport | null;
};
