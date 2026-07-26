import { NavigationContainer, DarkTheme } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { HomePage } from '@/modules/app';
import { MarketsPage } from '@/modules/markets';
import { colors, semanticColors } from '@/styles/tokens';
import type { HomeTabParamList, MainTabParamList, MarketsTabParamList } from './types';

const Tab = createBottomTabNavigator<MainTabParamList>();
const HomeStack = createNativeStackNavigator<HomeTabParamList>();
const MarketsStack = createNativeStackNavigator<MarketsTabParamList>();

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

function HomeStackNavigator() {
  return (
    <HomeStack.Navigator screenOptions={{ headerShown: false }}>
      <HomeStack.Screen name="Home" component={HomePage} />
    </HomeStack.Navigator>
  );
}

function MarketsStackNavigator() {
  return (
    <MarketsStack.Navigator screenOptions={{ headerShown: false }}>
      <MarketsStack.Screen name="MarketsList" component={MarketsPage} />
    </MarketsStack.Navigator>
  );
}

function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
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
    <NavigationContainer theme={navTheme}>
      <MainTabs />
    </NavigationContainer>
  );
}
