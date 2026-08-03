import { View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { SectionHeader } from '@/components/molecules/section-header';
import { DashboardMarketRow } from '@/components/organisms/dashboard-market-row';
import type { DashboardSectionListProps } from './DashboardSectionList.types';
import { styles } from './DashboardSectionList.styles';

export function DashboardSectionList({
  title,
  actionLabel,
  onAction,
  rows,
  isLoading,
  errorMessage,
  emptyMessage,
  onPressRow,
  onRetry,
  retryLabel = 'Retry',
}: DashboardSectionListProps) {
  return (
    <View style={styles.card}>
      <SectionHeader title={title} actionLabel={actionLabel} onAction={onAction} />
      <View style={styles.body}>
        {isLoading && rows.length === 0 ? (
          <View style={styles.skeletons}>
            <Skeleton height={40} width="100%" />
            <Skeleton height={40} width="100%" />
            <Skeleton height={40} width="90%" />
          </View>
        ) : errorMessage ? (
          <View style={styles.error}>
            <Text variant="caption" color="error">
              {errorMessage}
            </Text>
            {onRetry ? (
              <Button label={retryLabel} variant="secondary" onPress={onRetry} />
            ) : null}
          </View>
        ) : rows.length === 0 ? (
          <View style={styles.empty}>
            <Text variant="body" color="secondary">
              {emptyMessage ?? '—'}
            </Text>
          </View>
        ) : (
          rows.map((r) => (
            <DashboardMarketRow key={r.id} row={r} onPress={onPressRow} />
          ))
        )}
      </View>
    </View>
  );
}
