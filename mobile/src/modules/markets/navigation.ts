export const MarketsScreens = {
  List: 'MarketsList',
  Filters: 'MarketsFilters',
} as const;

export type MarketsStackParamList = {
  [MarketsScreens.List]: undefined;
  [MarketsScreens.Filters]: undefined;
};
