import type { ReactNode } from 'react';

export type PageEnterProps = {
  children: ReactNode;
  /** Route identity — change remounts the enter animation. */
  motionKey: string;
};
