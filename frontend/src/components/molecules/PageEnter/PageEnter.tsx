import { Stage } from './PageEnter.styles';
import type { PageEnterProps } from './PageEnter.types';

/** Route-level fade/slide-in. Remount via `motionKey` (pathname). */
export function PageEnter({ children, motionKey }: PageEnterProps) {
  return <Stage key={motionKey}>{children}</Stage>;
}
