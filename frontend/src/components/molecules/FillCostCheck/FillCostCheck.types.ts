/** Quote currency the comparison named, beside what the finished order paid. */
export type FillCostCheckView = {
  expectedFee?: number;
  actualFee?: number;
  feeDelta?: number;
  expectedSlippage?: number;
  actualSlippage?: number;
  slippageDelta?: number;
  quotedQuantity?: number;
  filledQuantity?: number;
};
