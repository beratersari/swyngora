import { FlatList, RefreshControl, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { MarketRow } from '@/components/organisms/market-row';
import { semanticColors } from '@/styles/tokens';
import type { MarketsListProps } from './MarketsList.types';
import { styles } from './MarketsList.styles';

function LoadMoreSkeleton() {
  return (
    <View style={styles.skeletonStack} accessibilityLabel="Loading more markets">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} height={72} width="100%" />
      ))}
    </View>
  );
}

export function MarketsList({
  rows,
  isLoading,
  isLoadingMore = false,
  hasMore = false,
  emptyMessage,
  errorMessage,
  onRetry,
  onPressRow,
  onLoadMore,
  ListHeaderComponent,
  refreshing = false,
  onRefresh,
  isWatched,
  onStarPress,
}: MarketsListProps) {
  if (isLoading && rows.length === 0) {
    return (
      <View style={styles.list}>
        {ListHeaderComponent}
        <View style={styles.skeletonStack}>
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} height={72} width="100%" />
          ))}
        </View>
      </View>
    );
  }

  return (
    <FlatList
      style={styles.list}
      contentContainerStyle={styles.content}
      data={rows}
      keyExtractor={(item) => item.id}
      renderItem={({ item }) => (
        <MarketRow
          row={item}
          onPress={onPressRow}
          watched={isWatched?.(item.symbol)}
          onStarPress={onStarPress}
        />
      )}
      ListHeaderComponent={ListHeaderComponent}
      ListFooterComponent={
        isLoadingMore ? (
          <LoadMoreSkeleton />
        ) : hasMore ? (
          <View style={styles.center}>
            <Text variant="caption" color="steel">
              Scroll for more
            </Text>
          </View>
        ) : rows.length > 0 ? (
          <View style={styles.center}>
            <Text variant="caption" color="steel">
              End of list
            </Text>
          </View>
        ) : null
      }
      ListEmptyComponent={
        <View style={styles.center}>
          {errorMessage ? (
            <>
              <Text variant="body" color="error">
                {errorMessage}
              </Text>
              <Button label="Retry" onPress={onRetry} />
            </>
          ) : (
            <Text variant="body" color="secondary">
              {emptyMessage ?? 'No markets match filters'}
            </Text>
          )}
        </View>
      }
      onEndReached={() => {
        if (hasMore && !isLoadingMore) {
          onLoadMore?.();
        }
      }}
      onEndReachedThreshold={0.35}
      refreshControl={
        onRefresh ? (
          <RefreshControl
            refreshing={refreshing}
            onRefresh={onRefresh}
            tintColor={semanticColors.text.primary}
            colors={[semanticColors.action.primary]}
          />
        ) : undefined
      }
    />
  );
}
