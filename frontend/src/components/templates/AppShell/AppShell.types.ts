import type { ReactNode } from 'react';

export type AppShellProps = {
  brand: ReactNode;
  nav: ReactNode;
  tools?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  navAriaLabel?: string;
  /** Wider content column for dense list pages (Markets, Watchlist, Pumps). */
  wide?: boolean;
  /** Optional strip below header (health / monitoring status). */
  banner?: ReactNode;
};
