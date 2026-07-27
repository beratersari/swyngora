export type RsiBadgeProps = {
  /** Preformatted label e.g. "RSI 62.4" or "—" / "…" */
  label: string;
  tone?: 'success' | 'warning' | 'error' | 'secondary' | 'steel';
  loading?: boolean;
  size?: 'sm' | 'md';
};
