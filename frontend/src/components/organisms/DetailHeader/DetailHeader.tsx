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
  formatDelistDay,
  formatSymbolDisplay,
  pairQuote,
} from '@/libs/utils';
import { useDisplayCurrency } from '@/libs/hooks';
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
  logoUrl,
  listingDate,
  contractLabel,
  backTo = '/markets',
  isLoading = false,
  watched = false,
  onToggleWatch,
  watchLoading = false,
  alertTo,
  compareTo,
  signalsTo,
  delistTime,
  announcedAt,
  halted = false,
}: DetailHeaderProps) {
  const { t } = useTranslation(['detail', 'watchlist', 'alerts', 'markets', 'signals']);
  const { formatPrice } = useDisplayCurrency();
  const delistLabel = formatDelistDate(delistTime);
  const announcedLabel = formatDelistDay(announcedAt);

  return (
    <HeaderCard>
      <TopRow>
        <TitleBlock>
          <BackLink to={backTo}>{t('backToMarkets')}</BackLink>
          <TitleRow>
            {logoUrl ? (
              <img
                src={logoUrl}
                alt=""
                width={28}
                height={28}
                style={{ borderRadius: 14, objectFit: 'contain' }}
              />
            ) : null}
            <Text variant="h2" color="primary" mono isLoading={isLoading} skeletonWidth={140}>
              {formatSymbolDisplay(symbol)}
            </Text>
            <BrandTag variant="exchange">{exchange}</BrandTag>
            {halted ? <BrandTag variant="delist">{t('detail:errors.tickerHalted')}</BrandTag> : null}
            {delistLabel ? (
              <BrandTag variant="delist">
                {announcedLabel
                  ? t('markets:table.delistTagAnnounced', {
                      date: delistLabel,
                      announced: announcedLabel,
                    })
                  : t('markets:table.delistTag', { date: delistLabel })}
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
            {listingDate ? (
              <Text variant="caption" color="secondary">
                {listingDate}
              </Text>
            ) : null}
            {contractLabel ? (
              <Text variant="caption" color="secondary" mono>
                {contractLabel}
              </Text>
            ) : null}
          </TitleRow>
        </TitleBlock>
        <PriceBlock>
          <FlashValue value={lastPrice}>
            <Text variant="h3" color="primary" mono isLoading={isLoading} skeletonWidth={120}>
              {formatPrice(lastPrice, pairQuote(symbol, exchange))}
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
