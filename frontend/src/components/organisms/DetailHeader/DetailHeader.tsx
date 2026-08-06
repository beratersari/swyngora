import { Button, Tag } from 'antd';
import { StarFilled, StarOutlined } from '@ant-design/icons';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import {
  changeTone,
  formatChangePercent,
  formatDelistDate,
  formatPrice,
  formatSymbolDisplay,
} from '@/libs/utils';
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
  alertTo,
  compareTo,
  delistTime,
}: DetailHeaderProps) {
  const { t } = useTranslation(['detail', 'watchlist', 'alerts', 'markets']);
  const delistLabel = formatDelistDate(delistTime);

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
            {delistLabel ? (
              <BrandTag variant="delist">
                {t('markets:table.delistTag', { date: delistLabel })}
              </BrandTag>
            ) : null}
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
            {alertTo ? (
              <Link to={alertTo}>
                <Button size="small" type="link">
                  {t('alerts:addFromDetail', { defaultValue: 'Add alert' })}
                </Button>
              </Link>
            ) : null}
            {compareTo ? (
              <Link to={compareTo}>
                <Button size="small" type="link">
                  {t('alerts:addToCompare', { defaultValue: 'Compare' })}
                </Button>
              </Link>
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
            mono
            isLoading={isLoading}
            skeletonWidth={72}
          >
            {formatChangePercent(priceChangePercent)}
          </Text>
        </PriceBlock>
      </TopRow>
    </HeaderCard>
  );
}
