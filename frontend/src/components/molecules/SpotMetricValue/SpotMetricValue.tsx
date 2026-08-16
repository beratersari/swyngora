import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { FlashValue } from '@/components/molecules/FlashValue';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  changeTone,
  formatChangePercent,
  formatCompactUsd,
  formatDelistDate,
  formatTradeCount,
  marketCapQuote,
  pairQuote,
} from '@/libs/utils';
import { TagsWrap } from './SpotMetricValue.styles';
import type { SpotMetricValueProps } from './SpotMetricValue.types';

/**
 * Renders one SpotMarket field using the shared metric catalog definition.
 */
export function SpotMetricValue({
  metric,
  spot,
  exchange,
  isLoading = false,
  locale: localeProp,
}: SpotMetricValueProps) {
  const { t, i18n } = useTranslation(['markets', 'common']);
  const { formatPrice: formatMoney, formatCompact } = useDisplayCurrency();
  const locale = localeProp ?? i18n.language;
  const raw = spot?.[metric.field];
  const quote = pairQuote(spot?.symbol, exchange);
  const mcapQuote = marketCapQuote(exchange);

  if (metric.format === 'tags') {
    const tags = (raw as string[] | undefined) ?? [];
    const delistLabel = formatDelistDate(spot?.delistTime, locale);
    // Backend injects synthetic "Delist" into tags[]; avoid duplicating plain Tag + BrandTag.
    const productTags = tags.filter((tag) => tag.toLowerCase() !== 'delist');
    if (isLoading) {
      return (
        <Text variant="caption" color="secondary" isLoading skeletonWidth={80}>
          —
        </Text>
      );
    }
    if (!productTags.length && !delistLabel) {
      return (
        <Text variant="caption" color="secondary">
          —
        </Text>
      );
    }
    return (
      <TagsWrap>
        {delistLabel ? (
          <BrandTag variant="delist">
            {t('markets:table.delistTag', { date: delistLabel })}
          </BrandTag>
        ) : null}
        {productTags.slice(0, delistLabel ? 3 : 4).map((tag) => (
          <BrandTag key={tag} variant="outline">
            {tag}
          </BrandTag>
        ))}
        {productTags.length > (delistLabel ? 3 : 4) ? (
          <BrandTag variant="paused">+{productTags.length - (delistLabel ? 3 : 4)}</BrandTag>
        ) : null}
      </TagsWrap>
    );
  }

  let display: string;
  switch (metric.format) {
    case 'price':
      display = formatMoney(raw as string | number | null | undefined, quote);
      break;
    case 'changePercent':
      display = formatChangePercent(raw as string | number | null | undefined);
      break;
    case 'compactUsd':
      display =
        metric.id === 'quoteVolume'
          ? formatCompact(raw as string | number | null | undefined, quote)
          : metric.id === 'marketCapCirculating' ||
              metric.id === 'marketCapTotal' ||
              metric.id === 'marketCapMax'
            ? formatCompact(raw as string | number | null | undefined, mcapQuote)
            : formatCompactUsd(raw as string | number | null | undefined);
      break;
    case 'tradeCount':
      display = formatTradeCount(raw as number | null | undefined, exchange);
      break;
    case 'number':
      display =
        raw == null || raw === ''
          ? '—'
          : typeof raw === 'number'
            ? raw.toLocaleString(locale)
            : String(raw);
      break;
    default:
      display = '—';
  }

  if (metric.format === 'changePercent' && !isLoading) {
    const tone = changeTone(raw as string | number | null | undefined);
    const variant = tone === 'success' ? 'up' : tone === 'error' ? 'down' : 'paused';
    return (
      <FlashValue value={raw}>
        <BrandTag variant={variant}>{display}</BrandTag>
      </FlashValue>
    );
  }

  const color = metric.toneFromChange
    ? changeTone(raw as string | number | null | undefined)
    : metric.format === 'price'
      ? 'primary'
      : 'secondary';

  return (
    <FlashValue value={raw}>
      <Text
        variant="numeric"
        color={color}
        isLoading={isLoading}
        skeletonWidth={metric.format === 'changePercent' ? 56 : 72}
      >
        {display}
      </Text>
    </FlashValue>
  );
}
