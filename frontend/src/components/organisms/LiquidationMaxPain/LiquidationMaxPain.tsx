import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import { pocketLevels, sideTone, venueLabel } from './helpers';
import {
  Banner,
  BannerTitle,
  CompareGrid,
  ExtraList,
  ExtraRow,
  Hint,
  MetricLabel,
  MetricTable,
  MetricValue,
  Panel,
  PriceNum,
  SideCard,
  SideChip,
  SideHead,
  VenueBlock,
  VenueHead,
  VenueMeta,
  VenueStack,
} from './LiquidationMaxPain.styles';
import type { LiquidationMaxPainProps, MaxPainPocket, MaxPainVenue } from './LiquidationMaxPain.types';

export function LiquidationMaxPain({ data, isLoading, errorMessage }: LiquidationMaxPainProps) {
  const { t } = useTranslation(['liquidations', 'common']);
  const { formatCompact, formatPrice } = useDisplayCurrency();
  const money = (v: string | number | null | undefined) => formatCompact(v, 'USDT');
  const px = (v: string | number | null | undefined) => formatPrice(v, 'USDT');
  const venues = data?.venues ?? [];

  if (isLoading && !data) {
    return (
      <Panel data-testid="liquidation-max-pain">
        <Skeleton height={88} />
        <Skeleton height={220} />
      </Panel>
    );
  }

  return (
    <Panel data-testid="liquidation-max-pain">
      {errorMessage ? <Alert type="error" showIcon message={errorMessage} /> : null}
      <Banner>
        <BannerTitle>{data?.summary || t('liquidations:maxPain.empty')}</BannerTitle>
        <Text variant="bodySm" color="secondary">
          {t('liquidations:maxPain.hint')}
        </Text>
      </Banner>
      {venues.length === 0 ? (
        <Text variant="bodySm" color="secondary">
          {t('liquidations:maxPain.pick')}
        </Text>
      ) : (
        <VenueStack>
          {venues.map((venue) => (
            <VenueCard key={venue.exchange ?? venue.symbol} venue={venue} money={money} formatPx={px} />
          ))}
        </VenueStack>
      )}
      <Hint>{data?.note || t('liquidations:disclaimer')}</Hint>
    </Panel>
  );
}

function VenueCard({
  venue,
  money,
  formatPx,
}: {
  venue: MaxPainVenue;
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
}) {
  const { t } = useTranslation(['liquidations', 'common']);
  return (
    <VenueBlock>
      <VenueHead>
        <Text variant="h4">{venueLabel(venue.exchange)}</Text>
        <VenueMeta>
          <Text variant="caption" color="secondary">
            {t('liquidations:maxPain.last')}: {formatPx(venue.price)}
          </Text>
          <Text variant="caption" color="secondary">
            {t('liquidations:maxPain.oi')}: {money(venue.openInterestValue)}
          </Text>
        </VenueMeta>
      </VenueHead>
      {venue.error ? <Alert type="warning" showIcon message={venue.error} /> : null}
      <CompareGrid>
        <PocketCard
          side="up"
          title={t('liquidations:maxPain.aboveTitle')}
          subtitle={t('liquidations:maxPain.aboveHint')}
          pocket={venue.above}
          extras={pocketLevels(venue.above, venue.aboveLevels)}
          money={money}
          formatPx={formatPx}
        />
        <PocketCard
          side="down"
          title={t('liquidations:maxPain.belowTitle')}
          subtitle={t('liquidations:maxPain.belowHint')}
          pocket={venue.below}
          extras={pocketLevels(venue.below, venue.belowLevels)}
          money={money}
          formatPx={formatPx}
        />
      </CompareGrid>
    </VenueBlock>
  );
}

function PocketCard({
  side,
  title,
  subtitle,
  pocket,
  extras,
  money,
  formatPx,
}: {
  side: 'up' | 'down';
  title: string;
  subtitle: string;
  pocket?: MaxPainPocket;
  extras: MaxPainPocket[];
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
}) {
  const { t } = useTranslation('liquidations');
  const tone = sideTone(pocket?.side) === 'down' ? 'down' : side;
  return (
    <SideCard $side={tone} data-testid={`liquidation-max-pain-${side}`}>
      <SideHead>
        <div>
          <Text variant="h4">{title}</Text>
          <Text variant="caption" color="secondary">
            {subtitle}
          </Text>
        </div>
        {pocket?.side ? <SideChip $side={tone}>{t(`maxPain.sides.${pocket.side === 'long' ? 'long' : 'short'}`)}</SideChip> : null}
      </SideHead>
      {pocket?.price ? (
        <>
          <PriceNum>{formatPx(pocket.price)}</PriceNum>
          <MetricTable>
            <MetricLabel>{t('maxPain.distance')}</MetricLabel>
            <MetricValue>{pocket.movePct || '—'}</MetricValue>
            <MetricLabel>{t('maxPain.size')}</MetricLabel>
            <MetricValue>{money(pocket.notional)}</MetricValue>
            {pocket.leverage ? (
              <>
                <MetricLabel>{t('maxPain.leverage')}</MetricLabel>
                <MetricValue>{`${pocket.leverage}x`}</MetricValue>
              </>
            ) : null}
          </MetricTable>
        </>
      ) : (
        <Text variant="bodySm" color="secondary">
          {t('maxPain.none')}
        </Text>
      )}
      {extras.length > 0 ? (
        <ExtraList>
          {extras.map((row) => (
            <ExtraRow key={`${row.price}-${row.notional}`}>
              <span>
                {formatPx(row.price)} {row.movePct ? `(${row.movePct})` : ''}
              </span>
              <span>{money(row.notional)}</span>
            </ExtraRow>
          ))}
        </ExtraList>
      ) : null}
    </SideCard>
  );
}
