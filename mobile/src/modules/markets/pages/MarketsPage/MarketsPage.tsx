import { View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { ExchangeChips } from '@/components/organisms/ExchangeChips';
import { MarketsList } from '@/components/organisms/MarketsList';
import { MarketsToolbar } from '@/components/organisms/MarketsToolbar';
import { ScreenTemplate } from '@/components/templates/ScreenTemplate';
import type { MarketsPageProps, MarketsPageViewModel } from './MarketsPage.types';
import { useMarketsPageViewModel } from './MarketsPage.viewModel';
import { styles } from './MarketsPage.styles';

function MarketsPageView({ vm }: { vm: MarketsPageViewModel }) {
  const header = (
    <View style={styles.headerBlock}>
      <ExchangeChips
        exchanges={vm.exchanges}
        selected={vm.selectedExchange}
        onSelect={vm.onSelectExchange}
        isLoading={vm.exchangesLoading}
      />
      <MarketsToolbar
        search={vm.search}
        onSearchChange={vm.onSearchChange}
        isSearchDebouncing={vm.isSearchDebouncing}
        activeFilterCount={vm.activeFilterCount}
        onOpenFilters={vm.onOpenFilters}
      />
      {vm.filterSummary ? (
        <Text variant="caption" color="steel" style={styles.hint}>
          Active: {vm.filterSummary}
        </Text>
      ) : null}
      {vm.summaryLabel ? (
        <Text variant="caption" color="secondary" style={styles.summary}>
          {vm.summaryLabel}
          {vm.isPollingPaused ? ' · live refresh paused' : ''}
        </Text>
      ) : null}
      {vm.detailHint ? (
        <Text variant="caption" color="steel" style={styles.hint}>
          {vm.detailHint}
        </Text>
      ) : null}
    </View>
  );

  return (
    <ScreenTemplate title={vm.title}>
      <MarketsList
        rows={vm.rows}
        isLoading={vm.isLoading}
        isLoadingMore={vm.isLoadingMore}
        hasMore={vm.hasMore}
        emptyMessage={vm.emptyMessage}
        errorMessage={vm.errorMessage}
        onRetry={vm.onRetry}
        onPressRow={vm.onPressRow}
        onLoadMore={vm.onLoadMore}
        ListHeaderComponent={header}
        refreshing={vm.isRefreshing}
        onRefresh={vm.onRefresh}
      />
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
