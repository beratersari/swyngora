export const MarketsScreens = {
  List: 'MarketsList',
} as const;

export type MarketsStackParamList = {
  [MarketsScreens.List]: undefined;
};
