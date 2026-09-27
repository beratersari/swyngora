export type SweepRecoveryRow = {
  exchange?: string;
  side?: string;
  level?: string;
  recovered?: boolean;
  recoverySeconds?: number;
  beforeNotional?: string;
  backNotional?: string;
  summary?: string;
};

export type SweepRecoveryReport = {
  summary?: string;
  medianRecoverySeconds?: number;
  rows?: SweepRecoveryRow[];
};

export type SweepRecoveryListProps = {
  report?: SweepRecoveryReport;
};
