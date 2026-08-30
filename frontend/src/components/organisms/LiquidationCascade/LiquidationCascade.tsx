import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  formatClock,
  formatDurationSec,
  formatRatio,
  gradeTone,
  orderedWindows,
  priceMoveTone,
  sideTone,
  venueLabel,
} from './helpers';
import {
  Banner,
  BannerTitle,
  BothNote,
  EpisodeTable,
  EpisodeWrap,
  GradeChip,
  HitButton,
  HitsTable,
  HitsWrap,
  Panel,
  SideChip,
  VenueCard,
  VenueGrid,
  VenueHead,
  WindowCell,
  WindowHead,
  WindowTable,
} from './LiquidationCascade.styles';
import type { LiquidationCascadeProps } from './LiquidationCascade.types';

export function LiquidationCascade({
  report,
  hits,
  isLoading,
  errorMessage,
  onOpenCoin,
}: LiquidationCascadeProps) {
  const { t } = useTranslation(['liquidations', 'common']);
  const { formatCompact } = useDisplayCurrency();
  const venues = report?.venues ?? [];
  const peak = venues.reduce((best, v) => {
    if (!best) return v;
    const rank = (g?: string) =>
      g === 'extreme' ? 4 : g === 'cascade' ? 3 : g === 'elevated' ? 2 : 0;
    return rank(v.grade) > rank(best.grade) ? v : best;
  }, venues[0]);
  const bannerTone = report?.both?.agree
    ? gradeTone(report.both.grade)
    : gradeTone(peak?.grade);

  if (isLoading && !report) {
    return (
      <Panel data-testid="liquidation-cascade">
        <Skeleton height={88} />
        <Skeleton height={220} />
      </Panel>
    );
  }

  return (
    <Panel data-testid="liquidation-cascade">
      {errorMessage ? <Alert type="error" showIcon message={errorMessage} /> : null}

      <Banner $tone={bannerTone}>
        <BannerTitle>{report?.summary || t('liquidations:cascade.empty')}</BannerTitle>
        {report?.both?.agree ? <BothNote>{report.both.summary}</BothNote> : null}
        {report?.both && !report.both.agree && report.both.summary ? (
          <Text variant="bodySm" color="secondary">
            {report.both.summary}
          </Text>
        ) : null}
      </Banner>

      <VenueGrid>
        {venues.map((venue) => {
          const tone = gradeTone(venue.grade);
          const side = sideTone(venue.side);
          return (
            <VenueCard key={`${venue.exchange}-${venue.symbol}`} $tone={tone}>
              <VenueHead>
                <Text variant="bodySm" weight={700}>
                  {venueLabel(venue.exchange)}
                </Text>
                <GradeChip $tone={tone}>
                  {t(`liquidations:cascade.grades.${tone}`)}
                </GradeChip>
              </VenueHead>
              <div>
                <SideChip $tone={side}>
                  {t(`liquidations:cascade.sides.${side}`)}
                </SideChip>
                {venue.hottest ? (
                  <Text variant="caption" color="secondary">
                    {' · '}
                    {venue.hottest}
                    {venue.score ? ` · ${Math.round(venue.score)}` : ''}
                  </Text>
                ) : null}
              </div>
              <Text variant="caption" color="secondary">
                {venue.summary}
              </Text>
              <WindowTable>
                <WindowHead>{t('liquidations:cascade.window')}</WindowHead>
                <WindowHead>{t('liquidations:cards.long')}</WindowHead>
                <WindowHead>{t('liquidations:cards.short')}</WindowHead>
                <WindowHead>{t('liquidations:cascade.ratio')}</WindowHead>
                <WindowHead>{t('liquidations:cascade.grade')}</WindowHead>
                {orderedWindows(venue.windows).map((w) => {
                  const hot = w.window === venue.hottest;
                  const wTone = gradeTone(w.grade);
                  return (
                    <span key={w.window} style={{ display: 'contents' }}>
                      <WindowCell $hot={hot}>{w.window}</WindowCell>
                      <WindowCell $hot={hot}>{formatCompact(w.longNotional, 'USDT')}</WindowCell>
                      <WindowCell $hot={hot}>{formatCompact(w.shortNotional, 'USDT')}</WindowCell>
                      <WindowCell $hot={hot}>{formatRatio(w.maxRatio)}</WindowCell>
                      <WindowCell $hot={hot}>
                        {t(`liquidations:cascade.grades.${wTone}`)}
                      </WindowCell>
                    </span>
                  );
                })}
              </WindowTable>
            </VenueCard>
          );
        })}
      </VenueGrid>

      {(report?.episodes ?? []).length > 0 ? (
        <EpisodeWrap>
          <Text variant="bodySm" weight={700}>
            {t('liquidations:cascade.episodesTitle')}
          </Text>
          <EpisodeTable>
            <WindowHead>{t('liquidations:cascade.started')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.duration')}</WindowHead>
            <WindowHead>{t('liquidations:exchange')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.side')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.gradeLabel')}</WindowHead>
            <WindowHead>{t('liquidations:cards.long')}</WindowHead>
            <WindowHead>{t('liquidations:cards.short')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.price')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.status')}</WindowHead>
            {(report?.episodes ?? []).map((ep, i) => {
              const g = gradeTone(ep.grade);
              const s = sideTone(ep.side);
              const move = priceMoveTone(ep.priceChangePct);
              const venue = ep.combined ? t('liquidations:cascade.bothVenues') : venueLabel(ep.exchange);
              return (
                <span key={`${ep.exchange}-${ep.startedAt}-${i}`} style={{ display: 'contents' }}>
                  <WindowCell $hot={ep.open}>{formatClock(ep.startedAt)}</WindowCell>
                  <WindowCell>{formatDurationSec(ep.durationSec)}</WindowCell>
                  <WindowCell $hot={ep.combined}>{venue}</WindowCell>
                  <SideChip $tone={s}>{t(`liquidations:cascade.sides.${s}`)}</SideChip>
                  <GradeChip $tone={g}>{t(`liquidations:cascade.grades.${g}`)}</GradeChip>
                  <WindowCell>{formatCompact(ep.longNotional, 'USDT')}</WindowCell>
                  <WindowCell>{formatCompact(ep.shortNotional, 'USDT')}</WindowCell>
                  <SideChip $tone={move}>
                    {ep.priceChangePct ? `${ep.priceChangePct}%` : '—'}
                  </SideChip>
                  <WindowCell>
                    {ep.open ? t('liquidations:cascade.running') : t('liquidations:cascade.ended')}
                  </WindowCell>
                </span>
              );
            })}
          </EpisodeTable>
        </EpisodeWrap>
      ) : null}

      {hits && hits.length > 0 ? (
        <HitsWrap>
          <Text variant="bodySm" weight={700}>
            {t('liquidations:cascade.hitsTitle')}
          </Text>
          <HitsTable>
            <WindowHead>{t('liquidations:chart.coin')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.side')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.gradeLabel')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.score')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.hottest')}</WindowHead>
            <WindowHead>{t('liquidations:cascade.both')}</WindowHead>
            {hits.map((hit) => {
              const g = gradeTone(hit.grade);
              const s = sideTone(hit.side);
              return (
                <span key={hit.symbol} style={{ display: 'contents' }}>
                  {onOpenCoin && hit.symbol ? (
                    <HitButton type="button" onClick={() => onOpenCoin(hit.symbol!)}>
                      {hit.symbol.replace(/USDT$|USDC$/i, '') || hit.symbol}
                    </HitButton>
                  ) : (
                    <WindowCell $hot>{hit.symbol}</WindowCell>
                  )}
                  <SideChip $tone={s}>{t(`liquidations:cascade.sides.${s}`)}</SideChip>
                  <GradeChip $tone={g}>{t(`liquidations:cascade.grades.${g}`)}</GradeChip>
                  <WindowCell>{hit.score ? Math.round(hit.score) : '—'}</WindowCell>
                  <WindowCell>{hit.hottest || '—'}</WindowCell>
                  <WindowCell>{hit.both ? t('liquidations:cascade.yes') : '—'}</WindowCell>
                </span>
              );
            })}
          </HitsTable>
        </HitsWrap>
      ) : null}

      <Text variant="caption" color="secondary">
        {report?.note || t('liquidations:cascade.hint')}
      </Text>
    </Panel>
  );
}
