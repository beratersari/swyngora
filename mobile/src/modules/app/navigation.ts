export const AppScreens = {
  Home: 'Home',
} as const;

export type AppStackParamList = {
  [AppScreens.Home]: undefined;
};
