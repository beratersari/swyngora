import type { WithLoadingProps } from '@/components/types';
import type { PostDelistResponse } from '@/libs/api';

export type PostDelistPanelProps = WithLoadingProps & {
  view?: PostDelistResponse | null;
  error?: string | null;
  lastPrice?: string;
};
