import type { NavigatorScreenParams } from '@react-navigation/native';
import type { AppStackParamList } from '@/modules/app';
import type { MarketsStackParamList } from '@/modules/markets';
import type { WatchlistStackParamList } from '@/modules/watchlist';
import type { PumpsStackParamList } from '@/modules/pumps';

export type HomeTabParamList = AppStackParamList;
export type MarketsTabParamList = MarketsStackParamList;
export type WatchlistTabParamList = WatchlistStackParamList;
export type PumpsTabParamList = PumpsStackParamList;

export type MainTabParamList = {
  HomeTab: NavigatorScreenParams<HomeTabParamList> | undefined;
  MarketsTab: NavigatorScreenParams<MarketsTabParamList> | undefined;
  WatchlistTab: NavigatorScreenParams<WatchlistTabParamList> | undefined;
  PumpsTab: NavigatorScreenParams<PumpsTabParamList> | undefined;
};

export type RootStackParamList = {
  MainTabs: NavigatorScreenParams<MainTabParamList> | undefined;
};
