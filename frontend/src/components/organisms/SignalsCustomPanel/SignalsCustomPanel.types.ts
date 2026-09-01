import type { CustomSignalItem, CustomSignalProfile } from '@/libs/api';
import type { CustomSignalFactorId } from './constants';

export type CustomWeightDraftRow = {
  id: CustomSignalFactorId;
  enabled: boolean;
  pct: number;
};

export type CustomMixPreviewFactor = {
  id: CustomSignalFactorId;
  score: number;
  status: 'used' | 'missing' | 'disabled';
  defaultPct: number;
  appliedPct: number;
  defaultEffect: number;
  appliedEffect: number;
  deltaEffect: number;
  detail?: string;
};

export type CustomMixPreviewRow = {
  exchange: string;
  symbol: string;
  interval?: string;
  defaultScore: number;
  appliedScore: number;
  delta: number;
  grade: string;
  factors: CustomMixPreviewFactor[];
  largestChange: CustomMixPreviewFactor | null;
  huntMissing: boolean;
  price?: number;
};

export type SignalsCustomPanelProps = {
  profiles: CustomSignalProfile[];
  selectedId: string;
  draft: CustomWeightDraftRow[];
  items: CustomSignalItem[];
  loading?: boolean;
  saving?: boolean;
  scanning?: boolean;
  saveError?: unknown;
  onSelectProfile: (id: string) => void;
  onDraft: (next: CustomWeightDraftRow[]) => void;
  onSave: (name: string) => void;
  onSaveAs: (name: string) => void;
  onDelete: () => void;
  onApply: () => void;
  onReset: () => void;
  onOpen: (exchange: string, symbol: string) => void;
};
