export const MarketsScreens = {
  List: 'MarketsList',
  Filters: 'MarketsFilters',
  Categories: 'Categories',
  Detail: 'CoinDetail',
} as const;

export type MarketsStackParamList = {
  [MarketsScreens.List]: undefined;
  [MarketsScreens.Filters]: undefined;
  [MarketsScreens.Categories]: undefined;
  [MarketsScreens.Detail]: {
    exchange: string;
    symbol: string;
  };
};
