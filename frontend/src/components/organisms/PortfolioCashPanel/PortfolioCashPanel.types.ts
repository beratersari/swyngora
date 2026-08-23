export type PortfolioCashPanelProps = {
  isDepositing?: boolean;
  isWithdrawing?: boolean;
  /** When true, deposit/withdraw is blocked (e.g. 2+ books and none selected). */
  disabled?: boolean;
  depositError?: unknown;
  withdrawError?: unknown;
  onDeposit: (amount: number, note?: string) => Promise<void>;
  onWithdraw: (amount: number, note?: string) => Promise<void>;
};
