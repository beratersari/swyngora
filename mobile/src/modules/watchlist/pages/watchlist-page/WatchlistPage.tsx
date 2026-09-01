import { FlatList, RefreshControl, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { WatchlistRow } from '@/components/organisms/watchlist-row';
import { ScreenTemplate } from '@/components/templates/screen-template';
import {
  useGetTicker24hQuery,
  type MarketExchange,
} from '@/libs/api';
import { useRefetchOnResume } from '@/libs/hooks';
import {
  changeTone,
  formatChangePercent,
  formatPrice,
  isMarketExchange,
  rtkCurrent,
  watchKey,
  type RsiRowFields,
} from '@/libs/utils';
import { WATCHLIST_QUOTE_POLL_MS } from '@/config/watchlistConstants';
import { semanticColors } from '@/styles/tokens';
import type { WatchlistPageProps, WatchlistPageViewModel } from './WatchlistPage.types';
import { useWatchlistPageViewModel } from './WatchlistPage.viewModel';
import { styles } from './WatchlistPage.styles';

/** Page-local: RTK quote for one pair (keeps Atomic organisms pure). */
function EnrichedWatchlistRow({
  exchange,
  symbol,
  pollQuotes,
  rsi,
  onPress,
  onUnstar,
}: {
  exchange: string;
  symbol: string;
  pollQuotes: boolean;
  rsi?: RsiRowFields;
  onPress: (exchange: string, symbol: string) => void;
  onUnstar: (exchange: string, symbol: string) => void;
}) {
  const ex: MarketExchange = isMarketExchange(exchange) ? exchange : 'binance';
  const query = useGetTicker24hQuery(
    { exchange: ex, symbol },
    {
      pollingInterval: pollQuotes ? WATCHLIST_QUOTE_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );
  useRefetchOnResume(query.refetch, pollQuotes);
  const ticker = rtkCurrent(query);

  return (
    <WatchlistRow
      row={{
        id: watchKey(exchange, symbol),
        exchange,
        symbol,
        lastPriceLabel: formatPrice(ticker?.lastPrice),
        changePercentLabel: formatChangePercent(ticker?.priceChangePercent),
        changeTone: changeTone(ticker?.priceChangePercent),
        quoteLoading: query.isLoading || query.isFetching,
        rsiLabel: rsi?.rsiLabel,
        rsiTone: rsi?.rsiTone,
        rsiLoading: rsi?.rsiLoading,
      }}
      onPress={onPress}
      onUnstar={onUnstar}
    />
  );
}

function WatchlistPageView({ vm }: { vm: WatchlistPageViewModel }) {
  const { t } = useTranslation(['watchlist', 'common']);
  const header = (
    <View style={styles.headerBlock}>
      {vm.countLabel ? (
        <Text variant="caption" color="secondary">
          {vm.countLabel}
          {vm.isPollingPaused ? ` · ${t('common:status.liveRefreshPaused')}` : ''}
        </Text>
      ) : null}
      {vm.actionError ? (
        <Text variant="caption" color="error" style={styles.hint}>
          {vm.actionError}
        </Text>
      ) : null}
      {vm.indicatorsError ? (
        <View style={styles.hint}>
          <Text variant="caption" color="error">
            {vm.indicatorsError}
          </Text>
          <Button
            label={t('common:actions.retryIndicators')}
            variant="secondary"
            onPress={vm.onRetry}
          />
        </View>
      ) : null}
      {vm.indicatorsDisclaimer ? (
        <Text variant="caption" color="steel" style={styles.hint}>
          {vm.indicatorsDisclaimer}
        </Text>
      ) : null}
      {vm.emptyMessage ? (
        <Button
          label={t('watchlist:openMarkets')}
          variant="secondary"
          onPress={vm.onOpenMarkets}
        />
      ) : null}
    </View>
  );

  if (vm.isLoading && vm.pairs.length === 0) {
    return (
      <ScreenTemplate title={vm.title}>
        {header}
        <View style={{ gap: 8 }}>
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} height={72} width="100%" />
          ))}
        </View>
      </ScreenTemplate>
    );
  }

  return (
    <ScreenTemplate title={vm.title}>
      <FlatList
        data={vm.pairs}
        keyExtractor={(item) => watchKey(item.exchange, item.symbol)}
        contentContainerStyle={{ gap: 8, paddingBottom: 24, flexGrow: 1 }}
        ListHeaderComponent={header}
        ListEmptyComponent={
          <View style={{ padding: 16, alignItems: 'center', gap: 12 }}>
            {vm.errorMessage ? (
              <>
                <Text variant="body" color="error">
                  {vm.errorMessage}
                </Text>
                <Button label={t('common:actions.retry')} onPress={vm.onRetry} />
              </>
            ) : (
              <Text variant="body" color="secondary">
                {vm.emptyMessage ?? t('watchlist:emptyList')}
              </Text>
            )}
          </View>
        }
        renderItem={({ item }) => (
          <EnrichedWatchlistRow
            exchange={item.exchange}
            symbol={item.symbol}
            pollQuotes={vm.pollQuotes}
            rsi={vm.rsiByKey.get(watchKey(item.exchange, item.symbol))}
            onPress={vm.onPressRow}
            onUnstar={vm.onUnstar}
          />
        )}
        refreshControl={
          <RefreshControl
            refreshing={vm.isRefreshing}
            onRefresh={vm.onRefresh}
            tintColor={semanticColors.text.primary}
            colors={[semanticColors.action.primary]}
          />
        }
      />
    </ScreenTemplate>
  );
}

function WatchlistPageConnected() {
  const vm = useWatchlistPageViewModel();
  return <WatchlistPageView vm={vm} />;
}

export function WatchlistPage({ viewModel }: WatchlistPageProps = {}) {
  if (viewModel) {
    return <WatchlistPageView vm={viewModel} />;
  }
  return <WatchlistPageConnected />;
}
