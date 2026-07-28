import { View } from 'react-native';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import type { PumpEventListProps } from './PumpEventList.types';
import { styles } from './PumpEventList.styles';

export function PumpEventList({
  title,
  subtitle,
  rows,
  isLoading,
  errorMessage,
  emptyMessage,
  disclaimer,
}: PumpEventListProps) {
  const { t } = useTranslation('detail');
  const resolvedTitle = title ?? t('pumpsSection');
  return (
    <View style={styles.section}>
      <Text variant="label" color="secondary">
        {resolvedTitle}
      </Text>
      {subtitle ? (
        <Text variant="caption" color="steel">
          {subtitle}
        </Text>
      ) : null}

      {isLoading && rows.length === 0 ? (
        <Skeleton height={64} width="100%" />
      ) : errorMessage ? (
        <Text variant="caption" color="error">
          {errorMessage}
        </Text>
      ) : rows.length === 0 ? (
        <Text variant="caption" color="secondary">
          {emptyMessage ?? t('pumpEventsEmpty')}
        </Text>
      ) : (
        rows.map((row) => (
          <View key={row.id} style={styles.card}>
            <View style={styles.rowTop}>
              <Text variant="label" color={row.returnTone}>
                {row.returnLabel}
              </Text>
              <Text variant="caption" color="steel">
                {row.timeLabel}
              </Text>
            </View>
            {row.metaLabel ? (
              <Text variant="caption" color="secondary" numberOfLines={2}>
                {row.metaLabel}
              </Text>
            ) : null}
          </View>
        ))
      )}

      {disclaimer ? (
        <Text variant="caption" color="steel">
          {disclaimer}
        </Text>
      ) : null}
    </View>
  );
}
