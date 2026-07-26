import { View } from 'react-native';
import { Text } from '@/components/atoms/Text';
import { ScreenTemplate } from '@/components/templates/ScreenTemplate';
import { ExchangeChips } from '../../components/ExchangeChips';
import { MarketsFilterBar } from '../../components/MarketsFilterBar';
import { MarketsList } from '../../components/MarketsList';
import { MarketsPagination } from '../../components/MarketsPagination';
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
      <MarketsFilterBar
        search={vm.search}
        onSearchChange={vm.onSearchChange}
        quote={vm.quote}
        quoteOptions={vm.quoteOptions}
        onQuoteChange={vm.onQuoteChange}
        availableTags={vm.availableTags}
        selectedTags={vm.selectedTags}
        onToggleTag={vm.onToggleTag}
        onClearTags={vm.onClearTags}
        sort={vm.sort}
        order={vm.order}
        sortOptions={vm.sortOptions}
        onSortChange={vm.onSortChange}
        onOrderChange={vm.onOrderChange}
      />
      {vm.summaryLabel ? (
        <Text variant="caption" color="secondary" style={styles.summary}>
          {vm.summaryLabel}
          {vm.isPollingPaused ? ' · polling paused' : ''}
        </Text>
      ) : null}
      {vm.lastUpdatedLabel ? (
        <Text variant="caption" color="steel" style={styles.hint}>
          {vm.lastUpdatedLabel}
        </Text>
      ) : null}
    </View>
  );

  const footer = (
    <MarketsPagination
      offset={vm.offset}
      limit={vm.limit}
      total={vm.total}
      canPrev={vm.canPrev}
      canNext={vm.canNext}
      onPrev={vm.onPrevPage}
      onNext={vm.onNextPage}
    />
  );

  return (
    <ScreenTemplate title={vm.title}>
      <MarketsList
        rows={vm.rows}
        isLoading={vm.isLoading}
        emptyMessage={vm.emptyMessage}
        errorMessage={vm.errorMessage}
        onRetry={vm.onRetry}
        onPressRow={vm.onPressRow}
        ListHeaderComponent={header}
        ListFooterComponent={footer}
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
