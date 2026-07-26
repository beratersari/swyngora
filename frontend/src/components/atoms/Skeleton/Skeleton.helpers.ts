import type { CSSProperties } from 'react';
import { VARIANT_DEFAULTS } from './Skeleton.constants';
import type { SkeletonVariant } from './Skeleton.types';

export function resolveSkeletonStyle(
  variant: SkeletonVariant,
  width?: number | string,
  height?: number | string,
): CSSProperties {
  const defaults = VARIANT_DEFAULTS[variant];
  return {
    width: width ?? defaults.width,
    height: height ?? defaults.height,
    borderRadius: defaults.borderRadius,
  };
}
