import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { Text } from '@/components/atoms/text';
import { ChipGroup } from '@/components/molecules/chip-group';
import { ExchangeChips } from '@/components/organisms/exchange-chips';
import { MarketsList } from '@/components/organisms/markets-list';
import { ScreenTemplate } from '@/components/templates/screen-template';
import type {
  LeaderboardsPageProps,
  LeaderboardsPageViewModel,
} from './LeaderboardsPage.types';
import { useLeaderboardsPageViewModel } from './LeaderboardsPage.viewModel';
import { styles } from './LeaderboardsPage.styles';

function LeaderboardsPageView({ vm }: { vm: LeaderboardsPageViewModel }) {
  const { t } = useTranslation(['leaderboards', 'common']);
  const header = (
    <View style={styles.headerBlock}>
      <View style={styles.back}>
        <Button label={vm.backLabel} variant="secondary" onPress={vm.onBack} />
      </View>
      <View style={styles.block}>
        <ChipGroup
          options={vm.boardOptions.map((o) => ({
            value: o.value,
            label: o.label,
          }))}
          selected={vm.board}
          onSelect={vm.onSelectBoard}
          mode="single"
          shape="pill"
          horizontalScroll
        />
      </View>
      <ExchangeChips
        exchanges={vm.exchanges}
        selected={vm.selectedExchange}
        onSelect={vm.onSelectExchange}
        isLoading={vm.exchangesLoading}
      />
      <View style={styles.block}>
        <Text variant="caption" color="secondary">
          {t('leaderboards:quote')}
        </Text>
        <ChipGroup
          options={vm.quoteOptions.map((q) => ({ value: q, label: q }))}
          selected={vm.quote}
          onSelect={vm.onSelectQuote}
          mode="single"
          shape="box"
          horizontalScroll
        />
      </View>
      {vm.summaryLabel ? (
        <Text variant="caption" color="secondary" style={styles.hint}>
          {vm.summaryLabel}
          {vm.isPollingPaused
            ? ` · ${t('common:status.liveRefreshPaused')}`
            : ''}
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

function LeaderboardsPageConnected() {
  const vm = useLeaderboardsPageViewModel();
  return <LeaderboardsPageView vm={vm} />;
}

export function LeaderboardsPage({ viewModel }: LeaderboardsPageProps = {}) {
  if (viewModel) {
    return <LeaderboardsPageView vm={viewModel} />;
  }
  return <LeaderboardsPageConnected />;
}
