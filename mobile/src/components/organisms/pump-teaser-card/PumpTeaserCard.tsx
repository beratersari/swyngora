import { Pressable, View } from 'react-native';
import { Button } from '@/components/atoms/button';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import { SectionHeader } from '@/components/molecules/section-header';
import type { PumpTeaserCardProps } from './PumpTeaserCard.types';
import { styles } from './PumpTeaserCard.styles';

export function PumpTeaserCard({
  title,
  actionLabel,
  onAction,
  items,
  isLoading,
  errorMessage,
  emptyMessage,
  disclaimer,
  onPressItem,
  onRetry,
  retryLabel = 'Retry',
}: PumpTeaserCardProps) {
  return (
    <View style={styles.card}>
      <SectionHeader title={title} actionLabel={actionLabel} onAction={onAction} />
      {isLoading && items.length === 0 ? (
        <>
          <Skeleton height={36} width="100%" />
          <Skeleton height={36} width="90%" />
        </>
      ) : errorMessage ? (
        <>
          <Text variant="caption" color="error">
            {errorMessage}
          </Text>
          {onRetry ? (
            <Button label={retryLabel} variant="secondary" onPress={onRetry} />
          ) : null}
        </>
      ) : items.length === 0 ? (
        <Text variant="body" color="secondary">
          {emptyMessage ?? '—'}
        </Text>
      ) : (
        items.map((item) => (
          <Pressable
            key={item.id}
            accessibilityRole="button"
            onPress={() => onPressItem?.(item.exchange, item.symbol)}
            style={styles.item}
          >
            <View style={styles.left}>
              <Text variant="h4">{item.symbol}</Text>
              <Text variant="caption" color="steel">
                {item.metaLabel || item.exchange}
              </Text>
            </View>
            <View style={styles.right}>
              <Text variant="label" color={item.returnTone}>
                {item.returnLabel}
              </Text>
            </View>
          </Pressable>
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
