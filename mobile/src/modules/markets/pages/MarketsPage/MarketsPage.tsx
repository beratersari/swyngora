import { View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { ScreenTemplate } from '@/components/templates/ScreenTemplate';
import type { MarketsPageProps, MarketsPageViewModel } from './MarketsPage.types';
import { useMarketsPageViewModel } from './MarketsPage.viewModel';
import { styles } from './MarketsPage.styles';

function MarketsPageView({ vm }: { vm: MarketsPageViewModel }) {
  return (
    <ScreenTemplate title={vm.title}>
      <View style={styles.card}>
        <Text variant="body">{vm.subtitle}</Text>
        <Text variant="caption" color="secondary">
          Multi-exchange spot list will land in the mobile markets epic.
        </Text>
      </View>
    </ScreenTemplate>
  );
}

function MarketsPageConnected() {
  const vm = useMarketsPageViewModel();
  return <MarketsPageView vm={vm} />;
}

export function MarketsPage({ viewModel }: MarketsPageProps = {}) {
  if (viewModel) {
    return <MarketsPageView vm={viewModel} />;
  }
  return <MarketsPageConnected />;
}
