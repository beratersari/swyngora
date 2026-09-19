import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import { kindTone, richerLine, shapeTone, slopeLabel, trendLabel, venueLabel } from './helpers';
import {
  Cell,
  Chip,
  CurveBanner,
  CurveTable,
  Head,
  Panel,
  VenueCard,
  VenueGrid,
  VenueHead,
} from './BasisPanel.styles';
import type { BasisPanelProps, BasisVenue } from './BasisPanel.types';

export function BasisPanel({ report, isLoading, errorMessage }: BasisPanelProps) {
  const { t } = useTranslation(['basis', 'common']);
  const { formatPrice } = useDisplayCurrency();
  const price = (value: string | number | undefined) => formatPrice(value, 'USDT');
  const venues = report?.venues ?? [];

  if (isLoading && !report) {
    return (
      <Panel data-testid="basis-panel">
        <Skeleton height={160} />
      </Panel>
    );
  }

  return (
    <Panel data-testid="basis-panel">
      {errorMessage ? <Alert type="error" showIcon message={errorMessage} /> : null}
      {report?.agreement ? (
        <Text variant="bodySm" weight={700}>
          {report.agreement.title}
        </Text>
      ) : null}
      {report?.agreement?.summary ? (
        <Text variant="caption" color="secondary">
          {report.agreement.summary}
        </Text>
      ) : null}

      <VenueGrid>
        {venues.map((v) => (
          <VenueCard key={v.exchange ?? v.symbol}>
            <VenueHead>
              <span>{venueLabel(v.exchange)}</span>
              <Chip $tone={kindTone(v.last?.kind)}>
                {t(`basis:kinds.${kindTone(v.last?.kind)}`)}
              </Chip>
            </VenueHead>
            <Text variant="bodySm">
              {v.last?.deltaPct ? `${v.last.deltaPct}%` : '—'}{' '}
              <Text as="span" variant="caption" color="secondary">
                {t(`basis:trends.${trendLabel(v.trend)}`)}
              </Text>
            </Text>
            <Text variant="caption" color="secondary">
              {v.summary || v.error || t('basis:empty')}
            </Text>
          </VenueCard>
        ))}
      </VenueGrid>

      {venues.map((v) => (
        <CurveBlock key={`curve-${v.exchange}`} venue={v} formatPrice={price} />
      ))}

      <Text variant="caption" color="secondary">
        {report?.note || t('basis:curveHint')}
      </Text>
    </Panel>
  );
}

function CurveBlock({
  venue,
  formatPrice,
}: {
  venue: BasisVenue;
  formatPrice: (value: string | number | undefined) => string;
}) {
  const { t } = useTranslation(['basis', 'common']);
  const curve = venue.curve;
  if (!curve?.complete || !(curve.points ?? []).length) return null;
  const tone = shapeTone(curve.shape);
  return (
    <div>
      <Text variant="bodySm" weight={700}>
        {venueLabel(venue.exchange)} · {t('basis:curveTitle')}
      </Text>
      <CurveBanner $tone={tone} style={{ marginTop: 8 }}>
        <Text variant="bodySm" weight={700}>
          {t(`basis:${richerLine(curve.richer)}`)} · {t(`basis:shapes.${tone}`)} ·{' '}
          {t(`basis:slopes.${slopeLabel(curve.slope)}`)}
        </Text>
        <Text variant="caption" color="secondary">
          {curve.summary}
        </Text>
        {curve.near && curve.far ? (
          <Text variant="caption">
            {t('basis:near')} {curve.near.label} {formatPrice(curve.near.mark)} · {t('basis:far')}{' '}
            {curve.far.label} {formatPrice(curve.far.mark)} · {t('basis:spread')} {curve.spreadPct}%
          </Text>
        ) : null}
      </CurveBanner>
      <CurveTable style={{ marginTop: 10 }}>
        <Head>{t('basis:contract')}</Head>
        <Head>{t('basis:expiry')}</Head>
        <Head>{t('basis:dte')}</Head>
        <Head>{t('basis:mark')}</Head>
        <Head>{t('basis:vsIndex')}</Head>
        {(curve.points ?? []).map((p) => (
          <span key={p.symbol} style={{ display: 'contents' }}>
            <Cell $hot>{p.label || p.symbol}</Cell>
            <Cell>{p.kind === 'perp' ? '—' : p.label}</Cell>
            <Cell>{p.kind === 'perp' ? '—' : p.daysToExpiry}</Cell>
            <Cell>{formatPrice(p.mark)}</Cell>
            <Cell>{p.vsIndexPct ? `${p.vsIndexPct}%` : '—'}</Cell>
          </span>
        ))}
      </CurveTable>
    </div>
  );
}
