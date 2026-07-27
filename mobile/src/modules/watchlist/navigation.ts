export const WatchlistScreens = {
  List: 'WatchlistList',
  Detail: 'CoinDetail',
} as const;

export type WatchlistStackParamList = {
  [WatchlistScreens.List]: undefined;
  [WatchlistScreens.Detail]: {
    exchange: string;
    symbol: string;
  };
};
