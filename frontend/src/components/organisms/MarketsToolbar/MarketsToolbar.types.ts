import type { ReactNode } from 'react';

export type MarketsToolbarProps = {
  q: string;
  quote: string;
  tag: string;
  tags: string[];
  tagsLoading?: boolean;
  onQChange: (q: string) => void;
  onQuoteChange: (quote: string) => void;
  onTagChange: (tag: string) => void;
  /** Optional trailing controls (e.g. metric column picker). */
  trailing?: ReactNode;
};
