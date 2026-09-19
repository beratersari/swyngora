import type { BasisCurve } from '@/components/organisms/BasisPanel/BasisPanel.types';
import type { BASIS_CURVE_RANGES } from './constants';

export type BasisCurveHistoryRange = (typeof BASIS_CURVE_RANGES)[number];

export type BasisCurveHistoryProps = {
  curve?: BasisCurve | null;
  venueLabel?: string;
};
