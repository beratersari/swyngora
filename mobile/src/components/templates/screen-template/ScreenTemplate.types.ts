import type { ReactNode } from 'react';

export type ScreenTemplateProps = {
  title: string;
  children: ReactNode;
  footer?: ReactNode;
};
