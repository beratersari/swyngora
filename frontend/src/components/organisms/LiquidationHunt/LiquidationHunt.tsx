import { useState } from 'react';
import { Alert, Button, InputNumber, Modal, Segmented, Switch } from 'antd';
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
  defaultHuntWeightDraft,
  huntWeightTotal,
  parseHuntPanel,
  parseNum,
  pathLeverageLabel,
  pathStepTone,
  previewHuntMix,
  scoreValue,
  venueLabel,
} from './helpers';
import { HUNT_SCORE_FACTORS } from './constants';
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
  PreviewBlock,
  PreviewCard,
  PreviewFactorTable,
  PreviewHead,
  PreviewPair,
  PreviewScores,
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
  HuntMixPreview,
  HuntPanel,
  HuntScoreCompare,
  HuntScoreMix,
  HuntVenue,
  HuntWeightDraftRow,
  LiquidationHuntProps,
} from './LiquidationHunt.types';

export function LiquidationHunt({
  data,
  isLoading,
  errorMessage,
  panel,
  onPanelChange,
  weightDraft,
  onApplyWeights,
}: LiquidationHuntProps) {
  const { t } = useTranslation(['liquidations', 'common']);
  const { formatCompact, formatPrice } = useDisplayCurrency();
  const money = (v: string | number | null | undefined) => formatCompact(v, 'USDT');
  const px = (v: string | number | null | undefined) => formatPrice(v, 'USDT');
  const venues = data?.venues ?? [];
  const lean = leanTone(data?.bias?.lean);
  const [localPanel, setLocalPanel] = useState<HuntPanel>('compare');
  const [weightsOpen, setWeightsOpen] = useState(false);
  const [draft, setDraft] = useState<HuntWeightDraftRow[]>(() => weightDraft ?? defaultHuntWeightDraft());
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
        <Button
          size="small"
          data-testid="liquidation-hunt-weights"
          onClick={() => {
            setDraft(weightDraft ?? defaultHuntWeightDraft());
            setWeightsOpen(true);
          }}
        >
          {t('liquidations:hunt.weights.open')}
        </Button>
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
        <ScoreMixStrip mix={data?.scoreMix} compare={data?.scoreCompare} />
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

      <HuntWeightsModal
        open={weightsOpen}
        draft={draft}
        venues={venues}
        onDraft={setDraft}
        onClose={() => setWeightsOpen(false)}
        onApply={() => {
          onApplyWeights?.(draft);
          setWeightsOpen(false);
        }}
        onReset={() => {
          const next = defaultHuntWeightDraft();
          setDraft(next);
          onApplyWeights?.(null);
          setWeightsOpen(false);
        }}
      />
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
      {path?.stallsAtIndex ? (
        <Text variant="caption" color="secondary" data-testid={`liquidation-hunt-path-${side}-stall`}>
          {t('hunt.path.stallBanner', { n: path.stallsAtIndex })}
          {path.stallNote ? ` — ${path.stallNote}` : ''}
        </Text>
      ) : null}
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
  const roleKey = step.role && ['start', 'self', 'helped', 'stall', 'unreachable', 'missing', 'observed'].includes(step.role)
    ? step.role
    : tone;
  const strength = parseNum(step.strength);
  const zoneEst =
    step.zoneEst === 'observed'
      ? t('hunt.path.zoneEstObserved')
      : step.zoneEst === 'missing'
        ? t('hunt.path.zoneEstMissing')
        : step.zoneEst === 'model'
          ? t('hunt.path.zoneEstModel')
          : null;
  const remaining =
    step.selfFueling
      ? t('hunt.path.roles.self')
      : step.role === 'missing'
        ? t('hunt.path.roles.missing')
        : !step.remaining?.reachable && step.remaining?.exhausted
          ? t('hunt.path.remainingUnreached')
          : money(step.remaining?.notional);
  return (
    <PathStep $tone={tone} data-testid="liquidation-hunt-path-step">
      <PathStepHead>
        <PathIndex>
          {t('hunt.path.step', { n: step.index ?? 0 })}
          {lev ? ` · ${t('hunt.path.leverage', { lev: lev.replace('x', '') })}` : ''}
        </PathIndex>
        <PathChip $tone={tone}>{t(`hunt.path.roles.${roleKey}`)}</PathChip>
      </PathStepHead>
      <Text variant="bodySm">
        {formatPx(step.band?.price)}
        {step.movePct ? ` (${step.movePct})` : ''}
      </Text>
      {strength != null ? (
        <ScoreBar title={t('hunt.path.strength')}>
          <ScoreFill $side={step.band?.direction === 'down' ? 'down' : 'up'} $pct={strength} />
        </ScoreBar>
      ) : null}
      <PathMeta>
        <MetricLabel>{t('hunt.path.zoneLiq')}</MetricLabel>
        <MetricValue>{money(step.zoneNotional)}</MetricValue>
        {zoneEst ? (
          <>
            <MetricLabel>{t('hunt.path.zoneEst')}</MetricLabel>
            <MetricValue>{zoneEst}</MetricValue>
          </>
        ) : null}
        <MetricLabel>{t('hunt.path.fromLast')}</MetricLabel>
        <MetricValue>{money(step.standalone?.notional)}</MetricValue>
        {(step.index ?? 0) > 1 ? (
          <>
            <MetricLabel>{t('hunt.path.hop')}</MetricLabel>
            <MetricValue>{money(step.incremental?.notional)}</MetricValue>
            <MetricLabel>{t('hunt.path.afterCascade')}</MetricLabel>
            <MetricValue $tone={tone === 'self' || tone === 'helped' ? 'profit' : 'muted'}>{remaining}</MetricValue>
            {step.assistancePct != null && step.assistancePct !== '' ? (
              <>
                <MetricLabel>{t('hunt.path.strength')}</MetricLabel>
                <MetricValue>{`${Math.round(Number(step.assistancePct))}`}</MetricValue>
              </>
            ) : null}
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

function ScoreMixStrip({ mix, compare }: { mix?: HuntScoreMix; compare?: HuntScoreCompare }) {
  const { t } = useTranslation('liquidations');
  if (!mix) return null;
  const applied = compare?.applied;
  const base = compare?.default;
  return (
    <Text variant="caption" color="secondary" data-testid="liquidation-hunt-score-mix">
      {mix.note || t('hunt.weights.defaultNote')}
      {mix.source === 'custom' ? ` (${t('hunt.weights.custom')})` : ''}
      {mix.usedTotal != null && mix.requestedTotal != null
        ? ` · ${t('hunt.weights.usedOf', { used: Math.round(mix.usedTotal), total: Math.round(mix.requestedTotal) })}`
        : ''}
      {mix.source === 'custom' && applied && base ? (
        <>
          {' · '}
          {t('hunt.weights.vsDefault', {
            applied: `${Math.round(applied.upScore ?? 0)}/${Math.round(applied.downScore ?? 0)}`,
            base: `${Math.round(base.upScore ?? 0)}/${Math.round(base.downScore ?? 0)}`,
            delta: `${formatEffect(compare.delta?.upScore)} / ${formatEffect(compare.delta?.downScore)}`,
          })}
        </>
      ) : null}
    </Text>
  );
}

function HuntWeightsModal({
  open,
  draft,
  venues,
  onDraft,
  onClose,
  onApply,
  onReset,
}: {
  open: boolean;
  draft: HuntWeightDraftRow[];
  venues: HuntVenue[];
  onDraft: (next: HuntWeightDraftRow[]) => void;
  onClose: () => void;
  onApply: () => void;
  onReset: () => void;
}) {
  const { t } = useTranslation('liquidations');
  const total = huntWeightTotal(draft);
  const ok = Math.abs(total - 100) < 0.05;
  const preview = previewHuntMix(venues, draft);
  return (
    <Modal
      open={open}
      title={t('hunt.weights.title')}
      width={720}
      onCancel={onClose}
      footer={[
        <Button key="reset" onClick={onReset}>
          {t('hunt.weights.reset')}
        </Button>,
        <Button key="cancel" onClick={onClose}>
          {t('hunt.weights.cancel')}
        </Button>,
        <Button key="apply" type="primary" disabled={!ok} onClick={onApply} data-testid="liquidation-hunt-weights-apply">
          {t('hunt.weights.apply')}
        </Button>,
      ]}
    >
      <Text variant="bodySm" color="secondary">
        {t('hunt.weights.hint')}
      </Text>
      {HUNT_SCORE_FACTORS.map((factor) => {
        const row = draft.find((d) => d.id === factor.id) ?? { id: factor.id, enabled: false, pct: 0 };
        return (
          <div key={factor.id} style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 10 }}>
            <Switch
              size="small"
              checked={row.enabled}
              onChange={(on) =>
                onDraft(draft.map((d) => (d.id === factor.id ? { ...d, enabled: on, pct: on ? d.pct || factor.defaultPct : 0 } : d)))
              }
            />
            <Text variant="bodySm" style={{ flex: 1 }}>
              {t(`hunt.weights.factors.${factor.id}`)}
            </Text>
            <InputNumber
              min={0}
              max={100}
              value={row.enabled ? row.pct : 0}
              disabled={!row.enabled}
              addonAfter="%"
              style={{ width: 120 }}
              onChange={(value) =>
                onDraft(draft.map((d) => (d.id === factor.id ? { ...d, pct: Number(value) || 0 } : d)))
              }
            />
          </div>
        );
      })}
      <Text variant="bodySm" color={ok ? 'secondary' : 'error'} data-testid="liquidation-hunt-weights-total">
        {t('hunt.weights.total', { total: Math.round(total * 10) / 10 })}
        {!ok ? ` — ${t('hunt.weights.needHundred')}` : ''}
      </Text>
      {preview ? <HuntWeightPreview preview={preview} /> : null}
    </Modal>
  );
}

function HuntWeightPreview({ preview }: { preview: HuntMixPreview }) {
  const { t } = useTranslation('liquidations');
  return (
    <PreviewBlock data-testid="liquidation-hunt-weights-preview">
      <Text variant="bodySm">{t('hunt.weights.preview')}</Text>
      <Text variant="caption" color="secondary">
        {t('hunt.weights.previewHint')}
      </Text>
      <PreviewPair>
        <PreviewCard data-testid="liquidation-hunt-weights-preview-default">
          <Text variant="caption" color="secondary">
            {t('hunt.weights.defaultMix')}
          </Text>
          <PreviewScores>
            <MetricLabel>{t('hunt.weights.up')}</MetricLabel>
            <MetricValue $tone="up">{Math.round(preview.defaultUp)}</MetricValue>
            <MetricLabel>{t('hunt.weights.down')}</MetricLabel>
            <MetricValue $tone="down">{Math.round(preview.defaultDown)}</MetricValue>
          </PreviewScores>
        </PreviewCard>
        <PreviewCard data-testid="liquidation-hunt-weights-preview-custom">
          <Text variant="caption" color="secondary">
            {t('hunt.weights.draftMix')}
          </Text>
          <PreviewScores>
            <MetricLabel>{t('hunt.weights.up')}</MetricLabel>
            <MetricValue $tone="up">
              {Math.round(preview.appliedUp)}{' '}
              <Text variant="caption" color="secondary">
                {formatEffect(preview.upDelta)}
              </Text>
            </MetricValue>
            <MetricLabel>{t('hunt.weights.down')}</MetricLabel>
            <MetricValue $tone="down">
              {Math.round(preview.appliedDown)}{' '}
              <Text variant="caption" color="secondary">
                {formatEffect(preview.downDelta)}
              </Text>
            </MetricValue>
          </PreviewScores>
        </PreviewCard>
      </PreviewPair>
      <HuntWeightPreviewFactors
        side="up"
        factors={preview.upFactors}
        largest={preview.upLargestChange}
      />
      <HuntWeightPreviewFactors
        side="down"
        factors={preview.downFactors}
        largest={preview.downLargestChange}
      />
    </PreviewBlock>
  );
}

function HuntWeightPreviewFactors({
  side,
  factors,
  largest,
}: {
  side: 'up' | 'down';
  factors: HuntMixPreview['upFactors'];
  largest: HuntMixPreview['upLargestChange'];
}) {
  const { t } = useTranslation('liquidations');
  return (
    <div data-testid={`liquidation-hunt-weights-preview-factors-${side}`}>
      <Text variant="caption" color="secondary">
        {t(side === 'up' ? 'hunt.weights.upFactors' : 'hunt.weights.downFactors')}
      </Text>
      {largest ? (
        <Text variant="bodySm" data-testid={`liquidation-hunt-weights-preview-largest-${side}`}>
          {t('hunt.weights.largestChange', {
            factor: t(`hunt.weights.factors.${largest.id}`),
            value: formatEffect(largest.deltaEffect),
          })}
        </Text>
      ) : null}
      <PreviewFactorTable>
        <PreviewHead>{t('hunt.factors.title')}</PreviewHead>
        <PreviewHead>{t('hunt.weights.defaultMix')}</PreviewHead>
        <PreviewHead>{t('hunt.weights.draftMix')}</PreviewHead>
        <PreviewHead>{t('hunt.weights.delta')}</PreviewHead>
        {factors.map((factor) => (
          <span key={factor.id} style={{ display: 'contents' }}>
            <FactorName title={factor.status === 'missing' ? t('hunt.weights.missingKept', { pct: Math.round(factor.appliedPct) }) : undefined}>
              {t(`hunt.weights.factors.${factor.id}`)}
              {factor.status === 'missing' && factor.appliedPct > 0
                ? ` · ${t('hunt.weights.missingKept', { pct: Math.round(factor.appliedPct) })}`
                : ''}
            </FactorName>
            <FactorShare>{`${Math.round(factor.defaultPct)}% · ${formatEffect(factor.defaultEffect)}`}</FactorShare>
            <FactorShare>{`${Math.round(factor.appliedPct)}% · ${formatEffect(factor.appliedEffect)}`}</FactorShare>
            <FactorMeta $tone={effectTone(factor.deltaEffect)}>{formatEffect(factor.deltaEffect)}</FactorMeta>
          </span>
        ))}
      </PreviewFactorTable>
    </div>
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
