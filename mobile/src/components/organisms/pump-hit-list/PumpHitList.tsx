import { FlatList, RefreshControl, View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { PumpHitRow } from '@/components/organisms/pump-hit-row';
import { semanticColors } from '@/styles/tokens';
import type { PumpHitListProps } from './PumpHitList.types';
import { styles } from './PumpHitList.styles';

export function PumpHitList({
  rows,
  isLoading,
  emptyMessage,
  errorMessage,
  onRetry,
  onPressRow,
  ListHeaderComponent,
  refreshing = false,
  onRefresh,
}: PumpHitListProps) {
  const { t } = useTranslation(['pumps', 'common']);
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
      renderItem={({ item }) => <PumpHitRow row={item} onPress={onPressRow} />}
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
              {emptyMessage ?? t('pumps:emptyList')}
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
