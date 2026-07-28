import { Button, Tag } from 'antd';
import { StarFilled, StarOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { changeTone, formatChangePercent, formatPrice, formatSymbolDisplay } from '@/libs/utils';
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
  watched = false,
  onToggleWatch,
  watchLoading = false,
}: DetailHeaderProps) {
  const { t } = useTranslation(['detail', 'watchlist']);

  return (
    <HeaderCard>
      <TopRow>
        <TitleBlock>
          <BackLink to={backTo}>{t('backToMarkets')}</BackLink>
          <TitleRow>
            <Text variant="h2" color="primary" mono isLoading={isLoading} skeletonWidth={140}>
              {formatSymbolDisplay(symbol)}
            </Text>
            <Tag color="processing">{exchange}</Tag>
            {onToggleWatch ? (
              <Button
                type="text"
                size="small"
                loading={watchLoading}
                icon={watched ? <StarFilled /> : <StarOutlined />}
                onClick={onToggleWatch}
                aria-label={watched ? t('watchlist:remove') : t('watchlist:add')}
              />
            ) : null}
            {assetName ? (
              <Text variant="body" color="secondary">
                {assetName}
              </Text>
            ) : null}
          </TitleRow>
        </TitleBlock>
        <PriceBlock>
          <Text variant="h3" color="primary" mono isLoading={isLoading} skeletonWidth={120}>
            {formatPrice(lastPrice)}
          </Text>
          <Text
            variant="label"
            color={changeTone(priceChangePercent)}
            isLoading={isLoading}
            skeletonWidth={72}
          >
            {t('change24h', { change: formatChangePercent(priceChangePercent) })}
          </Text>
        </PriceBlock>
      </TopRow>
    </HeaderCard>
  );
}
