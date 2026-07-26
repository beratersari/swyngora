import type { SkeletonVariant } from './Skeleton.types';

export const DEFAULT_VARIANT: SkeletonVariant = 'text';

export const VARIANT_DEFAULTS: Record<
  SkeletonVariant,
  { width?: number | string; height?: number | string; borderRadius?: number }
> = {
  text: { width: '100%', height: 14, borderRadius: 4 },
  title: { width: '55%', height: 28, borderRadius: 6 },
  button: { width: 96, height: 36, borderRadius: 8 },
  avatar: { width: 40, height: 40, borderRadius: 999 },
  image: { width: '100%', height: 160, borderRadius: 8 },
  chart: { width: '100%', height: 280, borderRadius: 8 },
  card: { width: '100%', height: 120, borderRadius: 8 },
  input: { width: '100%', height: 36, borderRadius: 8 },
};
