import type { ReactNode } from 'react';

export type AppShellProps = {
  brand: ReactNode;
  nav: ReactNode;
  tools?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  navAriaLabel?: string;
  /** Kept for callers; workspace is always full-bleed. */
  wide?: boolean;
  /** Drop workspace padding (heatmap / chart boards). */
  flush?: boolean;
  /** Quote strip under the utility bar. */
  banner?: ReactNode;
};
