import { Skeleton as AntSkeleton } from 'antd';
import { DEFAULT_VARIANT } from './Skeleton.constants';
import { resolveSkeletonStyle } from './Skeleton.helpers';
import type { SkeletonProps } from './Skeleton.types';
import { SkeletonBlock, SkeletonChartBlock } from './Skeleton.styles';

/**
 * Design-system skeleton atom (styled-components).
 *
 * Modes:
 * 1. Standalone: `<Skeleton variant="chart" />`
 * 2. Wrapper: `<Skeleton isLoading={flag}>{content}</Skeleton>`
 * 3. Components accept `isLoading` and render this atom when true
 */
export function Skeleton({
  variant = DEFAULT_VARIANT,
  active = true,
  width,
  height,
  isLoading,
  children,
  className,
  style,
  rows = 1,
  'aria-label': ariaLabel = 'Loading',
}: SkeletonProps) {
  if (children !== undefined && children !== null) {
    if (!isLoading) {
      return <>{children}</>;
    }
  }

  if (children !== undefined && isLoading === undefined) {
    return <>{children}</>;
  }

  const box = resolveSkeletonStyle(variant, width, height);

  if (variant === 'avatar') {
    return (
      <AntSkeleton.Avatar
        active={active}
        size="large"
        className={className}
        style={{ ...box, ...style }}
        aria-label={ariaLabel}
      />
    );
  }

  if (variant === 'button') {
    return (
      <AntSkeleton.Button
        active={active}
        className={className}
        style={{ ...box, ...style }}
        aria-label={ariaLabel}
      />
    );
  }

  if (variant === 'input') {
    return (
      <AntSkeleton.Input
        active={active}
        className={className}
        style={{ ...box, ...style }}
        aria-label={ariaLabel}
      />
    );
  }

  if (variant === 'text' || variant === 'title') {
    return (
      <SkeletonBlock className={className} style={style}>
        <AntSkeleton
          active={active}
          title={variant === 'title'}
          paragraph={variant === 'text' ? { rows, width: width ?? '100%' } : false}
          aria-label={ariaLabel}
        />
      </SkeletonBlock>
    );
  }

  return (
    <SkeletonChartBlock
      className={className}
      style={{ ...box, ...style }}
      role="status"
      aria-busy="true"
      aria-label={ariaLabel}
    />
  );
}
