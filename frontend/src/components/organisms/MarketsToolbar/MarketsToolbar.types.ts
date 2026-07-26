export type MarketsToolbarProps = {
  q: string;
  quote: string;
  tag: string;
  tags: string[];
  tagsLoading?: boolean;
  onQChange: (q: string) => void;
  onQuoteChange: (quote: string) => void;
  onTagChange: (tag: string) => void;
};
