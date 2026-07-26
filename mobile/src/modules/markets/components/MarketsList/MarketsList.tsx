import { FlatList, RefreshControl, View } from 'react-native';
import { Button } from '@/components/atoms/Button';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { semanticColors } from '@/styles/tokens';
import { MarketRow } from '../MarketRow';
import type { MarketsListProps } from './MarketsList.types';
import { styles } from './MarketsList.styles';

export function MarketsList({
  rows,
  isLoading,
  emptyMessage,
  errorMessage,
  onRetry,
  onPressRow,
  ListHeaderComponent,
  ListFooterComponent,
  refreshing = false,
  onRefresh,
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
      renderItem={({ item }) => <MarketRow row={item} onPress={onPressRow} />}
      ListHeaderComponent={ListHeaderComponent}
      ListFooterComponent={ListFooterComponent}
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
