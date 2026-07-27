import { FlatList, RefreshControl, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { WatchlistRow } from '@/components/organisms/watchlist-row';
import { ScreenTemplate } from '@/components/templates/screen-template';
import {
  useGetTicker24hQuery,
  type MarketExchange,
} from '@/libs/api';
import {
  changeTone,
  formatChangePercent,
  formatPrice,
  isMarketExchange,
  watchKey,
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
  onPress,
  onUnstar,
}: {
  exchange: string;
  symbol: string;
  pollQuotes: boolean;
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

  return (
    <WatchlistRow
      row={{
        id: watchKey(exchange, symbol),
        exchange,
        symbol,
        lastPriceLabel: formatPrice(query.data?.lastPrice),
        changePercentLabel: formatChangePercent(query.data?.priceChangePercent),
        changeTone: changeTone(query.data?.priceChangePercent),
        quoteLoading: query.isLoading || query.isFetching,
      }}
      onPress={onPress}
      onUnstar={onUnstar}
    />
  );
}

function WatchlistPageView({ vm }: { vm: WatchlistPageViewModel }) {
  const header = (
    <View style={styles.headerBlock}>
      {vm.countLabel ? (
        <Text variant="caption" color="secondary">
          {vm.countLabel}
          {vm.isPollingPaused ? ' · live refresh paused' : ''}
        </Text>
      ) : null}
      {vm.actionError ? (
        <Text variant="caption" color="error" style={styles.hint}>
          {vm.actionError}
        </Text>
      ) : null}
      {vm.emptyMessage ? (
        <Button label="Open Markets" variant="secondary" onPress={vm.onOpenMarkets} />
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
                <Button label="Retry" onPress={vm.onRetry} />
              </>
            ) : (
              <Text variant="body" color="secondary">
                {vm.emptyMessage ?? 'No watched pairs yet'}
              </Text>
            )}
          </View>
        }
        renderItem={({ item }) => (
          <EnrichedWatchlistRow
            exchange={item.exchange}
            symbol={item.symbol}
            pollQuotes={vm.pollQuotes}
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
