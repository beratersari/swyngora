import type { NavigatorScreenParams } from '@react-navigation/native';
import type { AppStackParamList } from '@/modules/app';
import type { MarketsStackParamList } from '@/modules/markets';
import type { WatchlistStackParamList } from '@/modules/watchlist';

export type HomeTabParamList = AppStackParamList;
export type MarketsTabParamList = MarketsStackParamList;
export type WatchlistTabParamList = WatchlistStackParamList;

export type MainTabParamList = {
  HomeTab: NavigatorScreenParams<HomeTabParamList> | undefined;
  MarketsTab: NavigatorScreenParams<MarketsTabParamList> | undefined;
  WatchlistTab: NavigatorScreenParams<WatchlistTabParamList> | undefined;
};

export type RootStackParamList = {
  MainTabs: NavigatorScreenParams<MainTabParamList> | undefined;
};
