import { View } from 'react-native';
import { Text } from '@/components/atoms/text';
import { ExchangeChips } from '@/components/organisms/exchange-chips';
import { MarketsList } from '@/components/organisms/markets-list';
import { MarketsToolbar } from '@/components/organisms/markets-toolbar';
import { ScreenTemplate } from '@/components/templates/screen-template';
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
        favoritesOnly={vm.favoritesOnly}
        onToggleFavoritesOnly={vm.onToggleFavoritesOnly}
        favoritesCount={vm.favoritesCount}
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
      {vm.actionError ? (
        <Text variant="caption" color="error" style={styles.hint}>
          {vm.actionError}
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
        isWatched={vm.isWatched}
        onStarPress={vm.onStarPress}
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
