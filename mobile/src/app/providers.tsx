import type { ReactNode } from 'react';
import { Provider } from 'react-redux';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { StyleSheet } from 'react-native';
import { store } from '@/libs/api';
import { semanticColors } from '@/styles/tokens';
import { RootNavigator } from './navigation';

type ProvidersProps = {
  children?: ReactNode;
};

export function Providers({ children }: ProvidersProps) {
  return (
    <GestureHandlerRootView style={styles.root}>
      <Provider store={store}>
        <SafeAreaProvider>
          {children ?? <RootNavigator />}
        </SafeAreaProvider>
      </Provider>
    </GestureHandlerRootView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: semanticColors.bg.canvas,
  },
});
