import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  compareRows,
  coverageTone,
  easeTone,
  inputTone,
  leanTone,
  scoreValue,
  venueLabel,
} from './helpers';
import {
  Banner,
  BannerTitle,
  CompareGrid,
  CoverageChip,
  CoverageFill,
  CoverageMeter,
  CoverageRow,
  EaseChip,
  Hint,
  InputList,
  InputPill,
  MetricLabel,
  MetricTable,
  MetricValue,
  Panel,
  ReasonItem,
  ReasonList,
  ScoreBar,
  ScoreBlock,
  ScoreFill,
  ScoreNum,
  SideCard,
  SideHead,
  VenueBlock,
  VenueHead,
  VenueMeta,
  VenueStack,
} from './LiquidationHunt.styles';
import type {
  HuntCoverage,
  HuntDirectionScore,
  HuntVenue,
  LiquidationHuntProps,
} from './LiquidationHunt.types';

export function LiquidationHunt({ data, isLoading, errorMessage }: LiquidationHuntProps) {
  const { t } = useTranslation(['liquidations', 'common']);
  const { formatCompact, formatPrice } = useDisplayCurrency();
  const money = (v: string | number | null | undefined) => formatCompact(v, 'USDT');
  const px = (v: string | number | null | undefined) => formatPrice(v, 'USDT');
  const venues = data?.venues ?? [];
  const lean = leanTone(data?.bias?.lean);
  const metricLabels: Record<string, string> = {
    target: t('liquidations:hunt.metrics.target'),
    spot: t('liquidations:hunt.metrics.spot'),
    liq: t('liquidations:hunt.metrics.liq'),
    efficiency: t('liquidations:hunt.metrics.efficiency'),
    desk: t('liquidations:hunt.metrics.desk'),
    book: t('liquidations:hunt.metrics.book'),
  };

  if (isLoading && !data) {
    return (
      <Panel data-testid="liquidation-hunt">
        <Skeleton height={88} />
        <Skeleton height={260} />
      </Panel>
    );
  }

  return (
    <Panel data-testid="liquidation-hunt">
      {errorMessage ? <Alert type="error" showIcon message={errorMessage} /> : null}

      <Banner $tone={lean}>
        <BannerTitle>{data?.bias?.summary || t('liquidations:hunt.empty')}</BannerTitle>
        <CoverageStrip coverage={data?.coverage ?? data?.bias?.coverage} />
        <Text variant="bodySm" color="secondary">
          {t('liquidations:hunt.hint')}
        </Text>
      </Banner>

      {venues.length === 0 ? (
        <Text variant="bodySm" color="secondary">
          {t('liquidations:hunt.pick')}
        </Text>
      ) : (
        <VenueStack>
          {venues.map((venue) => (
            <VenueCard
              key={venue.exchange ?? venue.symbol}
              venue={venue}
              money={money}
              formatPx={px}
              metricLabels={metricLabels}
              winner={leanTone(venue.bias?.lean ?? data?.bias?.lean)}
            />
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
  metricLabels,
  winner,
}: {
  venue: HuntVenue;
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
  metricLabels: Record<string, string>;
  winner: 'up' | 'down' | 'even';
}) {
  const { t } = useTranslation(['liquidations', 'common']);
  const rows = compareRows(venue, money);
  const price = formatPx(venue.price);
  const oi = money(venue.openInterestValue);

  return (
    <VenueBlock>
      <VenueHead>
        <Text variant="h4">{venueLabel(venue.exchange)}</Text>
        <VenueMeta>
          <Text variant="caption" color="secondary">
            {t('liquidations:hunt.last')}: {price}
          </Text>
          <Text variant="caption" color="secondary">
            {t('liquidations:hunt.oi')}: {oi}
          </Text>
          {venue.fundingPayer && venue.fundingPayer !== 'none' ? (
            <Text variant="caption" color="secondary">
              {t('liquidations:hunt.fundingPayer', { side: venue.fundingPayer })}
            </Text>
          ) : null}
        </VenueMeta>
      </VenueHead>
      <CoverageStrip coverage={venue.coverage} excluded={!venue.coverage?.usable || Boolean(venue.error)} />
      {venue.error ? <Alert type="warning" showIcon message={venue.error} /> : null}
      <CompareGrid>
        <DirectionCard
          side="up"
          title={t('liquidations:hunt.upTitle')}
          subtitle={t('liquidations:hunt.upThesis')}
          score={venue.upScore}
          winner={winner === 'up' && venue.coverage?.usable !== false}
          rows={rows}
          metricLabels={metricLabels}
        />
        <DirectionCard
          side="down"
          title={t('liquidations:hunt.downTitle')}
          subtitle={t('liquidations:hunt.downThesis')}
          score={venue.downScore}
          winner={winner === 'down' && venue.coverage?.usable !== false}
          rows={rows}
          metricLabels={metricLabels}
        />
      </CompareGrid>
    </VenueBlock>
  );
}

function DirectionCard({
  side,
  title,
  subtitle,
  score,
  winner,
  rows,
  metricLabels,
}: {
  side: 'up' | 'down';
  title: string;
  subtitle: string;
  score?: HuntDirectionScore;
  winner: boolean;
  rows: ReturnType<typeof compareRows>;
  metricLabels: Record<string, string>;
}) {
  const { t } = useTranslation('liquidations');
  const value = scoreValue(score?.score);
  const level = easeTone(score?.level);
  return (
    <SideCard $side={side} $winner={winner} data-testid={`liquidation-hunt-${side}`}>
      <SideHead>
        <div>
          <Text variant="h4">{title}</Text>
          <Text variant="caption" color="secondary">
            {subtitle}
          </Text>
        </div>
        <ScoreBlock>
          <ScoreNum>{Math.round(value)}</ScoreNum>
          <EaseChip $tone={level}>{t(`hunt.levels.${level}`)}</EaseChip>
        </ScoreBlock>
      </SideHead>
      <ScoreBar>
        <ScoreFill $side={side} $pct={value} />
      </ScoreBar>
      <MetricTable>
        {rows.map((row) => (
          <span key={row.id} style={{ display: 'contents' }}>
            <MetricLabel>{metricLabels[row.id] ?? row.id}</MetricLabel>
            <MetricValue $tone={side === 'up' ? row.upTone : row.downTone}>
              {side === 'up' ? row.up : row.down}
            </MetricValue>
          </span>
        ))}
      </MetricTable>
      {(score?.reasons ?? []).length > 0 ? (
        <ReasonList>
          {(score?.reasons ?? []).map((reason) => (
            <ReasonItem key={reason}>{reason}</ReasonItem>
          ))}
        </ReasonList>
      ) : null}
    </SideCard>
  );
}

function CoverageStrip({ coverage, excluded }: { coverage?: HuntCoverage; excluded?: boolean }) {
  const { t } = useTranslation('liquidations');
  if (!coverage && !excluded) return null;
  const level = coverageTone(coverage?.level);
  const pct = scoreValue(coverage?.score);
  return (
    <div data-testid="liquidation-hunt-coverage">
      <CoverageRow>
        <CoverageChip $tone={level}>
          {t(`hunt.coverage.${level}`)}
          {coverage?.score != null ? ` ${Math.round(pct)}` : ''}
        </CoverageChip>
        <CoverageMeter>
          <CoverageFill $tone={level} $pct={pct} />
        </CoverageMeter>
        {excluded ? (
          <Text variant="caption" color="secondary">
            {t('hunt.coverage.excluded')}
          </Text>
        ) : null}
        {coverage?.summary ? (
          <Text variant="caption" color="secondary">
            {coverage.summary}
          </Text>
        ) : null}
      </CoverageRow>
      {coverage?.inputs && coverage.inputs.length > 0 ? (
        <InputList>
          {coverage.inputs.map((input) => (
            <InputPill key={input.id ?? input.label} $tone={inputTone(input.status)} title={input.detail}>
              {input.label || input.id}: {t(`hunt.coverage.status.${inputTone(input.status)}`)}
            </InputPill>
          ))}
        </InputList>
      ) : null}
    </div>
  );
}
