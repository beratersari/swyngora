import { Tag } from 'antd';
import type { SpotMarket } from '@/libs/api';
import { Text } from '@/components/atoms/Text';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
  formatTradeCount,
  type SpotMetricDef,
} from '@/libs/utils';

export type SpotMetricValueProps = {
  metric: SpotMetricDef;
  spot: SpotMarket | undefined | null;
  exchange?: string;
  isLoading?: boolean;
};

/**
 * Renders one SpotMarket field using the shared metric catalog definition.
 */
export function SpotMetricValue({ metric, spot, exchange, isLoading = false }: SpotMetricValueProps) {
  const raw = spot?.[metric.field];

  if (metric.format === 'tags') {
    const tags = (raw as string[] | undefined) ?? [];
    if (isLoading) {
      return (
        <Text variant="caption" color="secondary" isLoading skeletonWidth={80}>
          —
        </Text>
      );
    }
    if (!tags.length) {
      return (
        <Text variant="caption" color="secondary">
          —
        </Text>
      );
    }
    return (
      <span style={{ display: 'inline-flex', flexWrap: 'wrap', gap: 4, maxWidth: 200 }}>
        {tags.slice(0, 4).map((tag) => (
          <Tag key={tag}>{tag}</Tag>
        ))}
        {tags.length > 4 ? <Tag>+{tags.length - 4}</Tag> : null}
      </span>
    );
  }

  let display: string;
  switch (metric.format) {
    case 'price':
      display = formatPrice(raw as string | number | null | undefined);
      break;
    case 'changePercent':
      display = formatChangePercent(raw as string | number | null | undefined);
      break;
    case 'compactUsd':
      display = formatCompactUsd(raw as string | number | null | undefined);
      break;
    case 'tradeCount':
      display = formatTradeCount(raw as number | null | undefined, exchange);
      break;
    case 'number':
      display =
        raw == null || raw === ''
          ? '—'
          : typeof raw === 'number'
            ? raw.toLocaleString()
            : String(raw);
      break;
    default:
      display = '—';
  }

  const color = metric.toneFromChange
    ? changeTone(raw as string | number | null | undefined)
    : metric.format === 'price' || metric.format === 'changePercent'
      ? 'primary'
      : 'secondary';

  return (
    <Text
      variant="numeric"
      color={color}
      isLoading={isLoading}
      skeletonWidth={metric.format === 'changePercent' ? 56 : 72}
    >
      {display}
    </Text>
  );
}
