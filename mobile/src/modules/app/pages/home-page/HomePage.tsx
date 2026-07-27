import { View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { HomePageProps, HomePageViewModel } from './HomePage.types';
import { useHomePageViewModel } from './HomePage.viewModel';
import { styles } from './HomePage.styles';

function HomePageView({ vm }: { vm: HomePageViewModel }) {
  return (
    <ScreenTemplate title={vm.title}>
      <Text variant="body" color="secondary">
        Mobile client (Chrome). Use the bottom tabs: Markets · Pumps · Favorites.
      </Text>

      <View style={styles.card}>
        <Text variant="label" color="secondary">
          Features
        </Text>
        <View style={styles.actions}>
          <Button label="Open Markets" onPress={vm.onOpenMarkets} />
          <Button label="Open Pumps radar" onPress={vm.onOpenPumps} variant="secondary" />
        </View>
        <Text variant="caption" color="steel">
          Pumps scans top-volume pairs for rapid moves (may take a few seconds).
        </Text>
      </View>

      <View style={styles.card}>
        <View style={styles.row}>
          <Text variant="label" color="secondary">
            API base
          </Text>
          <Text variant="code">{vm.apiBaseUrlLabel}</Text>
        </View>

        <View style={styles.row}>
          <Text variant="label" color="secondary">
            Backend health
          </Text>
          {vm.isLoading && !vm.healthDetail ? (
            <Skeleton height={18} width="60%" />
          ) : (
            <Text
              variant="body"
              style={
                vm.healthStatus === 'ok'
                  ? styles.badgeOk
                  : vm.healthStatus === 'error'
                    ? styles.badgeError
                    : undefined
              }
            >
              {vm.healthStatus === 'ok'
                ? `OK${vm.healthDetail ? ` (${vm.healthDetail})` : ''}`
                : vm.healthStatus === 'error'
                  ? (vm.errorMessage ?? 'Error')
                  : 'Checking…'}
            </Text>
          )}
        </View>

        <Text variant="caption" color="secondary">
          Polling {vm.isPollingPaused ? 'paused (background)' : 'active'}
        </Text>

        <View style={styles.actions}>
          <Button label="Retry health" onPress={vm.onRetry} variant="secondary" />
        </View>
      </View>
    </ScreenTemplate>
  );
}

function HomePageConnected() {
  const vm = useHomePageViewModel();
  return <HomePageView vm={vm} />;
}

export function HomePage({ viewModel }: HomePageProps = {}) {
  if (viewModel) {
    return <HomePageView vm={viewModel} />;
  }
  return <HomePageConnected />;
}
