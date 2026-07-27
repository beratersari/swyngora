import { useEffect, type ComponentType } from 'react';
import { View, StyleSheet } from 'react-native';
import {
  NavigationContainer,
  DarkTheme,
  useNavigation,
} from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import type { BottomTabNavigationProp } from '@react-navigation/bottom-tabs';
import { HomePage } from '@/modules/app';
import {
  CoinDetailPage,
  MarketsFilterPage,
  MarketsPage,
  MarketsProvider,
  MarketsScreens,
} from '@/modules/markets';
import {
  WatchlistPage,
  WatchlistProvider,
  WatchlistScreens,
  useWatchlist,
} from '@/modules/watchlist';
import { PumpsScanPage, PumpsScreens } from '@/modules/pumps';
import { colors, semanticColors } from '@/styles/tokens';
import type {
  HomeTabParamList,
  MainTabParamList,
  MarketsTabParamList,
  PumpsTabParamList,
  WatchlistTabParamList,
} from './types';

const Tab = createBottomTabNavigator<MainTabParamList>();
const HomeStack = createNativeStackNavigator<HomeTabParamList>();
const MarketsStack = createNativeStackNavigator<MarketsTabParamList>();
const WatchlistStack = createNativeStackNavigator<WatchlistTabParamList>();
const PumpsStack = createNativeStackNavigator<PumpsTabParamList>();

const fill = StyleSheet.create({
  root: { flex: 1, height: '100%', width: '100%' },
  scene: { flex: 1, height: '100%', backgroundColor: colors.navy },
});

const navTheme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: colors.cream,
    background: colors.navy,
    card: colors.navy,
    text: colors.cream,
    border: semanticColors.border.default,
    notification: colors.indigo,
  },
};

const stackScreenOptions = {
  headerShown: false,
  contentStyle: fill.scene,
  animation: 'none' as const,
};

function HomeStackNavigator() {
  return (
    <HomeStack.Navigator screenOptions={stackScreenOptions}>
      <HomeStack.Screen name="Home" component={HomePage} />
    </HomeStack.Navigator>
  );
}

function MarketsStackNavigator() {
  return (
    <MarketsProvider>
      <MarketsStack.Navigator screenOptions={stackScreenOptions}>
        <MarketsStack.Screen name={MarketsScreens.List} component={MarketsPage} />
        <MarketsStack.Screen name={MarketsScreens.Filters} component={MarketsFilterPage} />
        <MarketsStack.Screen name={MarketsScreens.Detail} component={CoinDetailPage} />
      </MarketsStack.Navigator>
    </MarketsProvider>
  );
}

function withExitWhenNoFavorites<P extends object>(Screen: ComponentType<P>) {
  return function FavoritesGuardedScreen(props: P) {
    const { count, isReady } = useWatchlist();
    const navigation = useNavigation();

    useEffect(() => {
      if (isReady && count === 0) {
        const parent =
          navigation.getParent<BottomTabNavigationProp<MainTabParamList>>();
        parent?.navigate('MarketsTab');
      }
    }, [isReady, count, navigation]);

    return <Screen {...props} />;
  };
}

const FavoritesListScreen = withExitWhenNoFavorites(WatchlistPage);
const FavoritesDetailScreen = withExitWhenNoFavorites(CoinDetailPage);

function WatchlistStackNavigator() {
  return (
    <WatchlistStack.Navigator screenOptions={stackScreenOptions}>
      <WatchlistStack.Screen
        name={WatchlistScreens.List}
        component={FavoritesListScreen}
      />
      <WatchlistStack.Screen
        name={WatchlistScreens.Detail}
        component={FavoritesDetailScreen}
      />
    </WatchlistStack.Navigator>
  );
}

function PumpsStackNavigator() {
  return (
    <PumpsStack.Navigator screenOptions={stackScreenOptions}>
      <PumpsStack.Screen name={PumpsScreens.Scan} component={PumpsScanPage} />
      <PumpsStack.Screen name={PumpsScreens.Detail} component={CoinDetailPage} />
    </PumpsStack.Navigator>
  );
}

function MainTabsInner() {
  const { count, isReady } = useWatchlist();
  const showFavoritesTab = isReady && count > 0;

  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        sceneStyle: fill.scene,
        tabBarStyle: {
          backgroundColor: colors.navy,
          borderTopColor: semanticColors.border.default,
        },
        tabBarActiveTintColor: colors.cream,
        tabBarInactiveTintColor: colors.steel,
      }}
    >
      <Tab.Screen
        name="HomeTab"
        component={HomeStackNavigator}
        options={{ title: 'Home' }}
      />
      <Tab.Screen
        name="MarketsTab"
        component={MarketsStackNavigator}
        options={{ title: 'Markets' }}
      />
      <Tab.Screen
        name="PumpsTab"
        component={PumpsStackNavigator}
        options={{
          title: 'Pumps',
          tabBarLabel: 'Pumps',
          tabBarAccessibilityLabel: 'Pumps radar',
        }}
      />
      {showFavoritesTab ? (
        <Tab.Screen
          name="WatchlistTab"
          component={WatchlistStackNavigator}
          options={{
            title: 'Favorites',
            tabBarBadge: count,
            tabBarBadgeStyle: {
              backgroundColor: '#F5C542',
              color: colors.navy,
              fontSize: 11,
              fontWeight: '700',
            },
          }}
        />
      ) : null}
    </Tab.Navigator>
  );
}

function MainTabs() {
  return (
    <WatchlistProvider>
      <MainTabsInner />
    </WatchlistProvider>
  );
}

export function RootNavigator() {
  return (
    <View style={fill.root}>
      <NavigationContainer theme={navTheme}>
        <MainTabs />
      </NavigationContainer>
    </View>
  );
}
