import type { ReactNode } from 'react';

export type DeskEmptyProps = {
  title: ReactNode;
  hint?: ReactNode;
  extra?: ReactNode;
  className?: string;
};
