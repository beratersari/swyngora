export type PortfolioCashPanelBook = {
  id: string;
  name?: string;
};

export type PortfolioCashPanelProps = {
  isDepositing?: boolean;
  isWithdrawing?: boolean;
  isTransferring?: boolean;
  /** When true, deposit/withdraw is blocked (e.g. 2+ books and none selected). */
  disabled?: boolean;
  depositError?: unknown;
  withdrawError?: unknown;
  transferError?: unknown;
  transferBooks?: PortfolioCashPanelBook[];
  onDeposit: (amount: number, note?: string) => Promise<void>;
  onWithdraw: (amount: number, note?: string) => Promise<void>;
  onTransfer?: (toPortfolioId: string, amount: number, note?: string) => Promise<void>;
};
