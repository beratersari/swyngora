import type { NavigatorScreenParams } from '@react-navigation/native';
import type { AppStackParamList } from '@/modules/app';
import type { MarketsStackParamList } from '@/modules/markets';

export type HomeTabParamList = AppStackParamList;
export type MarketsTabParamList = MarketsStackParamList;

export type MainTabParamList = {
  HomeTab: NavigatorScreenParams<HomeTabParamList> | undefined;
  MarketsTab: NavigatorScreenParams<MarketsTabParamList> | undefined;
};

export type RootStackParamList = {
  MainTabs: NavigatorScreenParams<MainTabParamList> | undefined;
};
