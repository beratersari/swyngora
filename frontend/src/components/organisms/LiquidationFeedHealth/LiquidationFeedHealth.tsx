import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatClock, gapHours, venuesOf } from './LiquidationFeedHealth.helpers';
import { LiveDot, Meta, Strip, VenueCard, VenueHead } from './LiquidationFeedHealth.styles';
import type { LiquidationFeedHealthProps } from './LiquidationFeedHealth.types';

/** Last print, last socket payload, and recent gaps per venue. */
export function LiquidationFeedHealth({ feed }: LiquidationFeedHealthProps) {
  const { t } = useTranslation('liquidations');
  const venues = venuesOf(feed);
  if (venues.length === 0) return null;
  const missing = feed?.missing ?? [];
  return (
    <div>
      <Strip data-testid="liquidation-feed-health">
        {venues.map((v) => {
          const live = Boolean(v.live);
          const gaps = gapHours(v);
          return (
            <VenueCard key={v.exchange} $live={live}>
              <VenueHead>
                <Text variant="caption" color="primary">
                  {v.exchange === 'bybit' ? t('feed.bybit') : t('feed.binance')}
                </Text>
                <LiveDot $live={live} aria-hidden />
              </VenueHead>
              <Meta>
                <span>
                  {t('feed.status')}: {live ? t('status.live') : t('feed.down')}
                </span>
                <span>
                  {t('feed.lastPrint')}: {formatClock(v.lastEventAt)}
                </span>
                <span>
                  {t('feed.lastSeen')}: {formatClock(v.lastSeenAt)}
                </span>
                {gaps > 0 ? (
                  <span>
                    {t('feed.gaps', { hours: gaps.toFixed(1) })}
                  </span>
                ) : (
                  <span>{t('feed.noGaps')}</span>
                )}
              </Meta>
            </VenueCard>
          );
        })}
      </Strip>
      {missing.length > 0 ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginTop: 8 }}
          message={t('feed.missingVenues', { venues: missing.join(', ') })}
        />
      ) : null}
    </div>
  );
}