import { View, StyleSheet } from 'react-native';
import { NavigationContainer, DarkTheme } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { HomePage } from '@/modules/app';
import {
  CoinDetailPage,
  MarketsFilterPage,
  MarketsPage,
  MarketsProvider,
  MarketsScreens,
} from '@/modules/markets';
import { colors, semanticColors } from '@/styles/tokens';
import type { HomeTabParamList, MainTabParamList, MarketsTabParamList } from './types';

const Tab = createBottomTabNavigator<MainTabParamList>();
const HomeStack = createNativeStackNavigator<HomeTabParamList>();
const MarketsStack = createNativeStackNavigator<MarketsTabParamList>();

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

function MainTabs() {
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
    </Tab.Navigator>
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
