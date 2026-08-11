import { Button } from 'antd';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { FlashValue } from '@/components/molecules/FlashValue';
import { WatchStar } from '@/components/molecules/WatchStar';
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
  signalsTo,
  delistTime,
}: DetailHeaderProps) {
  const { t } = useTranslation(['detail', 'watchlist', 'alerts', 'markets', 'signals']);
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
            <BrandTag variant="exchange">{exchange}</BrandTag>
            {delistLabel ? (
              <BrandTag variant="delist">
                {t('markets:table.delistTag', { date: delistLabel })}
              </BrandTag>
            ) : null}
            {onToggleWatch ? (
              <WatchStar
                watched={watched}
                loading={watchLoading}
                addLabel={t('watchlist:add')}
                removeLabel={t('watchlist:remove')}
                onClick={onToggleWatch}
              />
            ) : null}
            {alertTo ? (
              <Link to={alertTo}>
                <Button size="small">{t('alerts:addFromDetail', { defaultValue: 'Add alert' })}</Button>
              </Link>
            ) : null}
            {compareTo ? (
              <Link to={compareTo}>
                <Button size="small">{t('alerts:addToCompare', { defaultValue: 'Compare' })}</Button>
              </Link>
            ) : null}
            {signalsTo ? (
              <Link to={signalsTo}>
                <Button size="small">{t('signals:title')}</Button>
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
          <FlashValue value={lastPrice}>
            <Text variant="h3" color="primary" mono isLoading={isLoading} skeletonWidth={120}>
              {formatPrice(lastPrice)}
            </Text>
          </FlashValue>
          <FlashValue value={priceChangePercent}>
            <Text
              variant="label"
              color={changeTone(priceChangePercent)}
              mono
              isLoading={isLoading}
              skeletonWidth={72}
            >
              {formatChangePercent(priceChangePercent)}
            </Text>
          </FlashValue>
        </PriceBlock>
      </TopRow>
    </HeaderCard>
  );
}
