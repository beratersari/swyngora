import type { ReactNode } from 'react';

export type PageHeaderProps = {
  title: string;
  subtitle?: string;
  eyebrow?: string;
  extra?: ReactNode;
};
