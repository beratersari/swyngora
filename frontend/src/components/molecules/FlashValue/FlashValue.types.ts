import type { ReactNode } from 'react';

export type FlashValueProps = {
  /** Raw tick used to detect direction (price, %, volume). */
  value: unknown;
  children: ReactNode;
  className?: string;
};
