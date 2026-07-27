import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { PumpHitList } from '@/components/organisms/pump-hit-list';
import { PumpScanFilters } from '@/components/organisms/pump-scan-filters';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type { PumpsScanPageProps, PumpsScanPageViewModel } from './PumpsScanPage.types';
import { usePumpsScanPageViewModel } from './PumpsScanPage.viewModel';
import { styles } from './PumpsScanPage.styles';

function PumpsScanPageView({ vm }: { vm: PumpsScanPageViewModel }) {
  const header = (
    <View style={styles.headerBlock}>
      <PumpScanFilters
        exchanges={vm.exchanges}
        selectedExchange={vm.selectedExchange}
        onSelectExchange={vm.onSelectExchange}
        exchangesLoading={vm.exchangesLoading}
        lookbackHours={vm.lookbackHours}
        lookbackOptions={vm.lookbackOptions}
        onSelectLookback={vm.onSelectLookback}
        minReturnPct={vm.minReturnPct}
        thresholdOptions={vm.thresholdOptions}
        onSelectThreshold={vm.onSelectThreshold}
        direction={vm.direction}
        directionOptions={vm.directionOptions}
        onSelectDirection={vm.onSelectDirection}
        summaryLabel={vm.summaryLabel}
      />
      <Text variant="caption" color="steel">
        {vm.disclaimer}
      </Text>
    </View>
  );

  return (
    <ScreenTemplate title={vm.title}>
      <PumpHitList
        rows={vm.rows}
        isLoading={vm.isLoading}
        emptyMessage={vm.emptyMessage}
        errorMessage={vm.errorMessage}
        onRetry={vm.onRetry}
        onPressRow={vm.onPressRow}
        ListHeaderComponent={header}
        refreshing={vm.isRefreshing}
        onRefresh={vm.onRefresh}
      />
    </ScreenTemplate>
  );
}

function PumpsScanPageConnected() {
  const vm = usePumpsScanPageViewModel();
  return <PumpsScanPageView vm={vm} />;
}

export function PumpsScanPage({ viewModel }: PumpsScanPageProps = {}) {
  if (viewModel) {
    return <PumpsScanPageView vm={viewModel} />;
  }
  return <PumpsScanPageConnected />;
}
