import type { OrderPreview } from '@/libs/api';
import type { PreviewWarningId } from './constants';

export type OrderPreviewProps = {
  preview?: OrderPreview | null;
  loading?: boolean;
  failed?: boolean;
};

export type PreviewRow = {
  id: string;
  labelKey: string;
  value: string;
};

export type { PreviewWarningId };
