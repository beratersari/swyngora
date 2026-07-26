import { Tag } from 'antd';
import { Text } from '@/components/atoms/Text';
import { changeTone, formatChangePercent, formatPrice } from '@/libs/utils';
import {
  BackLink,
  HeaderCard,
  PriceBlock,
  TitleBlock,
  TitleRow,
  TopRow,
} from './DetailHeader.styles';
import type { DetailHeaderProps } from './DetailHeader.types';

export function DetailHeader({
  symbol,
  exchange,
  lastPrice,
  priceChangePercent,
  assetName,
  backTo = '/markets',
  isLoading = false,
}: DetailHeaderProps) {
  return (
    <HeaderCard>
      <TopRow>
        <TitleBlock>
          <BackLink to={backTo}>← Markets</BackLink>
          <TitleRow>
            <Text variant="h2" color="cream" mono isLoading={isLoading} skeletonWidth={140}>
              {symbol}
            </Text>
            <Tag color="processing">{exchange}</Tag>
            {assetName ? (
              <Text variant="body" color="steel">
                {assetName}
              </Text>
            ) : null}
          </TitleRow>
        </TitleBlock>
        <PriceBlock>
          <Text variant="h3" color="cream" mono isLoading={isLoading} skeletonWidth={120}>
            {formatPrice(lastPrice)}
          </Text>
          <Text
            variant="label"
            color={changeTone(priceChangePercent)}
            isLoading={isLoading}
            skeletonWidth={72}
          >
            {formatChangePercent(priceChangePercent)} · 24h
          </Text>
        </PriceBlock>
      </TopRow>
    </HeaderCard>
  );
}
