export const MarketsScreens = {
  List: 'MarketsList',
  Filters: 'MarketsFilters',
  Categories: 'Categories',
  Leaderboards: 'Leaderboards',
  Detail: 'CoinDetail',
} as const;

export type MarketsStackParamList = {
  [MarketsScreens.List]: undefined;
  [MarketsScreens.Filters]: undefined;
  [MarketsScreens.Categories]: undefined;
  [MarketsScreens.Leaderboards]: {
    board?: 'gainers' | 'losers' | 'volume';
  } | undefined;
  [MarketsScreens.Detail]: {
    exchange: string;
    symbol: string;
  };
};
