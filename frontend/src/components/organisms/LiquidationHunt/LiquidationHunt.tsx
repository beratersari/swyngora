import { useState } from 'react';
import { Alert, Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import {
  compareRows,
  coverageTone,
  easeTone,
  effectTone,
  formatEffect,
  inputSpanText,
  inputTone,
  leanTone,
  parseHuntPanel,
  pathLeverageLabel,
  pathStepTone,
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
  FactorList,
  FactorMeta,
  FactorName,
  FactorRow,
  FactorShare,
  Hint,
  InputList,
  InputPill,
  MetricLabel,
  MetricTable,
  MetricValue,
  Panel,
  PanelSwitch,
  PathChip,
  PathIndex,
  PathList,
  PathMeta,
  PathStep,
  PathStepHead,
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
  HuntCascadePath,
  HuntCascadeStep,
  HuntCoverage,
  HuntDirectionScore,
  HuntPanel,
  HuntVenue,
  LiquidationHuntProps,
} from './LiquidationHunt.types';

export function LiquidationHunt({
  data,
  isLoading,
  errorMessage,
  panel,
  onPanelChange,
}: LiquidationHuntProps) {
  const { t } = useTranslation(['liquidations', 'common']);
  const { formatCompact, formatPrice } = useDisplayCurrency();
  const money = (v: string | number | null | undefined) => formatCompact(v, 'USDT');
  const px = (v: string | number | null | undefined) => formatPrice(v, 'USDT');
  const venues = data?.venues ?? [];
  const lean = leanTone(data?.bias?.lean);
  const [localPanel, setLocalPanel] = useState<HuntPanel>('compare');
  const activePanel = panel ?? localPanel;
  const setPanel = (next: HuntPanel) => {
    if (onPanelChange) onPanelChange(next);
    else setLocalPanel(next);
  };
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

      <PanelSwitch>
        <Segmented
          size="small"
          value={activePanel}
          onChange={(next) => setPanel(parseHuntPanel(String(next)))}
          options={[
            { value: 'compare', label: t('liquidations:hunt.panels.compare') },
            { value: 'path', label: t('liquidations:hunt.panels.path') },
          ]}
        />
      </PanelSwitch>

      <Banner $tone={lean}>
        <BannerTitle>{data?.bias?.summary || t('liquidations:hunt.empty')}</BannerTitle>
        <CoverageStrip coverage={data?.coverage ?? data?.bias?.coverage} />
        <Text variant="bodySm" color="secondary">
          {t(activePanel === 'path' ? 'liquidations:hunt.path.hint' : 'liquidations:hunt.hint')}
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
              panel={activePanel}
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
  panel,
}: {
  venue: HuntVenue;
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
  metricLabels: Record<string, string>;
  winner: 'up' | 'down' | 'even';
  panel: HuntPanel;
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
      {panel === 'path' ? (
        <CompareGrid>
          <PathCard
            side="up"
            title={t('liquidations:hunt.upTitle')}
            path={venue.upCascade}
            money={money}
            formatPx={formatPx}
          />
          <PathCard
            side="down"
            title={t('liquidations:hunt.downTitle')}
            path={venue.downCascade}
            money={money}
            formatPx={formatPx}
          />
        </CompareGrid>
      ) : (
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
      )}
    </VenueBlock>
  );
}

function PathCard({
  side,
  title,
  path,
  money,
  formatPx,
}: {
  side: 'up' | 'down';
  title: string;
  path?: HuntCascadePath;
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
}) {
  const { t } = useTranslation('liquidations');
  const steps = path?.steps ?? [];
  return (
    <SideCard $side={side} data-testid={`liquidation-hunt-path-${side}`}>
      <SideHead>
        <div>
          <Text variant="h4">{title}</Text>
          <Text variant="caption" color="secondary">
            {path?.summary || t('hunt.path.empty')}
          </Text>
        </div>
      </SideHead>
      {steps.length === 0 ? (
        <Text variant="bodySm" color="secondary">
          {t('hunt.path.empty')}
        </Text>
      ) : (
        <PathList>
          {steps.map((step) => (
            <PathStepRow key={step.index ?? step.band?.price} step={step} money={money} formatPx={formatPx} />
          ))}
        </PathList>
      )}
    </SideCard>
  );
}

function PathStepRow({
  step,
  money,
  formatPx,
}: {
  step: HuntCascadeStep;
  money: (v: string | number | null | undefined) => string;
  formatPx: (v: string | number | null | undefined) => string;
}) {
  const { t } = useTranslation('liquidations');
  const tone = pathStepTone(step);
  const lev = pathLeverageLabel(step);
  const chip =
    tone === 'self'
      ? t('hunt.path.selfFueling')
      : tone === 'easier'
        ? t('hunt.path.easier')
        : tone === 'unreachable'
          ? t('hunt.path.unreachable')
          : t('hunt.path.stillNeeds');
  return (
    <PathStep $tone={tone} data-testid="liquidation-hunt-path-step">
      <PathStepHead>
        <PathIndex>
          {t('hunt.path.step', { n: step.index ?? 0 })}
          {lev ? ` · ${t('hunt.path.leverage', { lev: lev.replace('x', '') })}` : ''}
        </PathIndex>
        <PathChip $tone={tone}>{chip}</PathChip>
      </PathStepHead>
      <Text variant="bodySm">
        {formatPx(step.band?.price)}
        {step.movePct ? ` (${step.movePct})` : ''}
      </Text>
      <PathMeta>
        <MetricLabel>{t('hunt.path.zoneLiq')}</MetricLabel>
        <MetricValue>{money(step.zoneNotional)}</MetricValue>
        <MetricLabel>{t('hunt.path.fromLast')}</MetricLabel>
        <MetricValue>{money(step.standalone?.notional)}</MetricValue>
        {(step.index ?? 0) > 1 ? (
          <>
            <MetricLabel>{t('hunt.path.hop')}</MetricLabel>
            <MetricValue>{money(step.incremental?.notional)}</MetricValue>
            <MetricLabel>{t('hunt.path.afterCascade')}</MetricLabel>
            <MetricValue $tone={tone === 'easier' || tone === 'self' ? 'profit' : 'muted'}>
              {step.selfFueling ? t('hunt.path.selfFueling') : money(step.remaining?.notional)}
            </MetricValue>
          </>
        ) : null}
      </PathMeta>
      {step.note ? (
        <Text variant="caption" color="secondary">
          {step.note}
        </Text>
      ) : null}
    </PathStep>
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
      {(score?.factors ?? []).length > 0 ? (
        <FactorList data-testid={`liquidation-hunt-${side}-factors`}>
          {(score?.factors ?? []).map((factor) => (
            <FactorRow key={factor.id ?? factor.label} title={factor.detail}>
              <FactorName>{factor.label || factor.id}</FactorName>
              <FactorMeta $tone={effectTone(factor.effect)}>
                {formatEffect(factor.effect)}
              </FactorMeta>
              <FactorShare>
                {t('hunt.factors.share', { pct: Math.round(scoreValue(factor.sharePct)) })}
                {' · '}
                {t('hunt.factors.effect', { value: formatEffect(factor.effect) })}
              </FactorShare>
            </FactorRow>
          ))}
        </FactorList>
      ) : null}
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
              {input.label || input.id}:{' '}
              {inputSpanText(input.have, input.need, input.coverPct, input.age, input.stale) ||
                t(`hunt.coverage.status.${inputTone(input.status)}`)}
            </InputPill>
          ))}
        </InputList>
      ) : null}
    </div>
  );
}
