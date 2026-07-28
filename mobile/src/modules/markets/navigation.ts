export const MarketsScreens = {
  List: 'MarketsList',
  Filters: 'MarketsFilters',
  Detail: 'CoinDetail',
} as const;

export type MarketsStackParamList = {
  [MarketsScreens.List]: undefined;
  [MarketsScreens.Filters]: undefined;
  [MarketsScreens.Detail]: {
    exchange: string;
    symbol: string;
  };
};
