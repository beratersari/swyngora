import { Pressable, View } from 'react-native';
import { Skeleton } from '@/components/atoms/skeleton';
import { Text } from '@/components/atoms/text';
import type { CrossExchangeRowModel } from '@/libs/utils';
import type { CrossExchangeCompareProps } from './CrossExchangeCompare.types';
import { styles } from './CrossExchangeCompare.styles';

function Row({
  row,
  isCheapest,
  sourceLabel,
  cheapestLabel,
  unavailableLabel,
  onPress,
}: {
  row: CrossExchangeRowModel;
  isCheapest: boolean;
  sourceLabel: string;
  cheapestLabel: string;
  unavailableLabel: string;
  onPress?: (exchange: string, symbol: string) => void;
}) {
  const pressable =
    Boolean(onPress) &&
    !row.isSource &&
    row.status === 'ok' &&
    Boolean(row.symbol) &&
    row.symbol !== '—';

  const body = (
    <>
      <View style={styles.left}>
        <Text variant="h4" style={{ textTransform: 'capitalize' } as never}>
          {row.exchange}
        </Text>
        <Text variant="caption" color="steel">
          {row.symbol}
        </Text>
        <View style={styles.badges}>
          {row.isSource ? (
            <View style={styles.badge}>
              <Text variant="caption" color="cream">
                {sourceLabel}
              </Text>
            </View>
          ) : null}
          {isCheapest && row.status === 'ok' ? (
            <View style={[styles.badge, styles.badgeMuted]}>
              <Text variant="caption" color="secondary">
                {cheapestLabel}
              </Text>
            </View>
          ) : null}
        </View>
      </View>
      <View style={styles.right}>
        {row.status === 'loading' ? (
          <>
            <Skeleton width={72} height={16} />
            <Skeleton width={48} height={12} />
          </>
        ) : row.status === 'ok' ? (
          <>
            <Text variant="numeric">{row.lastPriceLabel}</Text>
            <Text variant="caption" color={row.changeTone}>
              {row.changePercentLabel}
            </Text>
            <Text variant="caption" color="steel">
              {row.quoteVolumeLabel}
            </Text>
          </>
        ) : (
          <Text variant="caption" color={row.status === 'error' ? 'error' : 'steel'}>
            {row.errorMessage || unavailableLabel}
          </Text>
        )}
      </View>
    </>
  );

  const rowStyle = [
    styles.row,
    row.isSource && styles.rowSource,
    isCheapest && row.status === 'ok' && styles.rowCheapest,
  ];

  if (pressable) {
    return (
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${row.exchange} ${row.symbol}`}
        onPress={() => onPress?.(row.exchange, row.symbol)}
        style={rowStyle}
      >
        {body}
      </Pressable>
    );
  }

  return <View style={rowStyle}>{body}</View>;
}

export function CrossExchangeCompare({
  title,
  rows,
  disclaimer,
  emptyMessage,
  cheapestId = null,
  unavailableLabel = 'Unavailable',
  sourceLabel = 'This venue',
  cheapestLabel = 'Lowest',
  onPressRow,
}: CrossExchangeCompareProps) {
  if (rows.length === 0 && emptyMessage) {
    return (
      <View style={styles.root}>
        <Text variant="label" color="secondary">
          {title}
        </Text>
        <Text variant="body" color="secondary">
          {emptyMessage}
        </Text>
      </View>
    );
  }

  if (rows.length === 0) return null;

  return (
    <View style={styles.root}>
      <Text variant="label" color="secondary">
        {title}
      </Text>
      <View style={styles.list}>
        {rows.map((row) => (
          <Row
            key={row.id}
            row={row}
            isCheapest={cheapestId === row.id}
            sourceLabel={sourceLabel}
            cheapestLabel={cheapestLabel}
            unavailableLabel={unavailableLabel}
            onPress={onPressRow}
          />
        ))}
      </View>
      {disclaimer ? (
        <Text variant="caption" color="steel">
          {disclaimer}
        </Text>
      ) : null}
    </View>
  );
}
