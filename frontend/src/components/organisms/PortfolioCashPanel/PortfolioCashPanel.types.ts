export type PortfolioCashPanelProps = {
  isDepositing?: boolean;
  isWithdrawing?: boolean;
  depositError?: unknown;
  withdrawError?: unknown;
  onDeposit: (amount: number, note?: string) => Promise<void>;
  onWithdraw: (amount: number, note?: string) => Promise<void>;
};
