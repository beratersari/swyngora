export const SITUATION_MATCH_CLAUSES = [
  { id: 'priceCvd', kinds: ['price_up_cvd_down', 'price_down_cvd_up', 'agrees'] },
  { id: 'openInterest', kinds: ['opened', 'closed', 'flat'] },
  { id: 'liquidation', kinds: ['long', 'short', 'none'] },
  { id: 'prints', kinds: ['long', 'short', 'even', 'quiet'] },
  { id: 'funding', kinds: ['long', 'short', 'none'] },
  { id: 'crowd', kinds: ['top_more_long', 'top_more_short', 'even'] },
  {
    id: 'size',
    kinds: [
      'whale_buy_retail_sell',
      'whale_sell_retail_buy',
      'both_buy',
      'both_sell',
      'whale_buy',
      'whale_sell',
      'retail_buy',
      'retail_sell',
      'flat',
    ],
  },
] as const;

export type SituationClauseId = (typeof SITUATION_MATCH_CLAUSES)[number]['id'];

export type SituationMatch = Partial<Record<SituationClauseId, string>>;
