import { FlatList, RefreshControl, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { WatchlistRow } from '@/components/organisms/watchlist-row';
import { semanticColors } from '@/styles/tokens';
import type { WatchlistListProps } from './WatchlistList.types';
import { styles } from './WatchlistList.styles';

export function WatchlistList({
  rows,
  isLoading,
  emptyMessage,
  errorMessage,
  onRetry,
  onPressRow,
  onUnstar,
  ListHeaderComponent,
  refreshing = false,
  onRefresh,
}: WatchlistListProps) {
  const { t } = useTranslation(['watchlist', 'common']);
  if (isLoading && rows.length === 0) {
    return (
      <View style={styles.list}>
        {ListHeaderComponent}
        <View style={styles.skeletonStack}>
          {Array.from({ length: 5 }).map((_, i) => (
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
        <WatchlistRow row={item} onPress={onPressRow} onUnstar={onUnstar} />
      )}
      ListHeaderComponent={ListHeaderComponent}
      ListEmptyComponent={
        <View style={styles.center}>
          {errorMessage ? (
            <>
              <Text variant="body" color="error">
                {errorMessage}
              </Text>
              <Button label={t('common:actions.retry')} onPress={onRetry} />
            </>
          ) : (
            <Text variant="body" color="secondary">
              {emptyMessage ?? t('watchlist:emptyList')}
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
