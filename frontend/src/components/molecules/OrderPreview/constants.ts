export const PREVIEW_WARNING_IDS = [
  'blocked',
  'dailyLoss',
  'weight',
  'exposure',
  'cash',
  'quantity',
  'borrow',
  'stopBeyondLiquidation',
  'book',
] as const;

export type PreviewWarningId = (typeof PREVIEW_WARNING_IDS)[number];
