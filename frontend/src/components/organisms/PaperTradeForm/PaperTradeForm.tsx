import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Checkbox, InputNumber, Select, Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { situationLines } from '@/components/molecules/CoinSituation';
import { ExecutionCompare, type ExecutionStyleId } from '@/components/molecules/ExecutionCompare';
import { OrderPreview } from '@/components/molecules/OrderPreview';
import { OrderReasonHistory } from '@/components/organisms/OrderReasonHistory';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage } from '@/libs/api';
import { sizedQuantity } from '@/libs/hooks/orderPreview';
import { Actions, Field, FieldRow, FormStack } from './PaperTradeForm.styles';
import { TakeProfitTargets } from './TakeProfitTargets';
import type { PaperTradeFormProps, PaperTradeFormValues } from './PaperTradeForm.types';
import { targetWire, targetsComplete, type TargetRow } from './targets';
import {
  canSizeByRisk,
  needsSide,
  needsTif,
  needsTpSl,
  needsTrail,
  needsTrigger,
  orderKeepsReason,
  presentReasonClauses,
  reasonClauseReport,
  toApiOrderType,
  type ReasonSituation,
  type SpotOrderKind,
} from './helpers';

/**
 * Spot paper order ticket: market, limit, stop, trailing, OCO, bracket.
 * Page owns RTK mutation.
 */
export function PaperTradeForm({
  lockedExchange,
  lockedSymbol,
  defaultExchange = 'binance',
  defaultSymbol = '',
  defaultSide = 'buy',
  showLotMethod = true,
  advanced = true,
  isSubmitting = false,
  disabled = false,
  submitError,
  compact = false,
  preview,
  previewLoading = false,
  previewFailed = false,
  execution,
  executionLoading = false,
  executionRefreshing = false,
  executionUpdatedAt = null,
  executionFailed = false,
  onDraftChange,
  onSubmit,
}: PaperTradeFormProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const locked = Boolean(lockedExchange && lockedSymbol);
  const [exchange, setExchange] = useState(String(lockedExchange ?? defaultExchange));
  const [symbol, setSymbol] = useState(lockedSymbol ?? defaultSymbol);
  const [side, setSide] = useState<'buy' | 'sell'>(defaultSide);
  const [kind, setKind] = useState<SpotOrderKind>('market');
  const [quantity, setQuantity] = useState<number | null>(null);
  const [triggerPrice, setTriggerPrice] = useState<number | null>(null);
  const [targets, setTargets] = useState<TargetRow[]>([
    { percent: 100, offsetPercent: null, price: null },
  ]);
  const [stopLossPrice, setStopLossPrice] = useState<number | null>(null);
  const [moveStopToEntry, setMoveStopToEntry] = useState(false);
  const [riskPercent, setRiskPercent] = useState<number | null>(null);
  const [trailType, setTrailType] = useState<'percent' | 'offset'>('percent');
  const [trailValue, setTrailValue] = useState<number | null>(null);
  const [lotMethod, setLotMethod] = useState<'fifo' | 'lifo'>('fifo');
  const [tif, setTif] = useState<'gtc' | 'ioc' | 'fok'>('gtc');
  const [pickedReason, setPickedReason] = useState<string[] | null>(null);
  const [bestCost, setBestCost] = useState(false);
  const [spread, setSpread] = useState(false);
  const [spreadMinutes, setSpreadMinutes] = useState<number | null>(10);
  const [sliceCount, setSliceCount] = useState<number | null>(10);
  const [maxSlippagePct, setMaxSlippagePct] = useState<number | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);
  const [localBusy, setLocalBusy] = useState(false);
  const inFlightRef = useRef(false);
  const busy = isSubmitting || localBusy;

  useEffect(() => {
    if (lockedExchange) setExchange(String(lockedExchange));
  }, [lockedExchange]);
  useEffect(() => {
    if (lockedSymbol) setSymbol(lockedSymbol);
  }, [lockedSymbol]);
  useEffect(() => {
    setPickedReason(null);
  }, [exchange, symbol, kind, side]);

  const kindOptions = useMemo(() => {
    const base: { value: SpotOrderKind; label: string }[] = [
      { value: 'market', label: t('portfolio:trade.kinds.market') },
      { value: 'limit', label: t('portfolio:trade.kinds.limit') },
      { value: 'stop_loss', label: t('portfolio:trade.kinds.stop') },
    ];
    if (advanced && !compact) {
      base.push(
        { value: 'trailing_stop', label: t('portfolio:trade.kinds.trailing') },
        { value: 'oco', label: t('portfolio:trade.kinds.oco') },
        { value: 'bracket', label: t('portfolio:trade.kinds.bracket') },
      );
    }
    return base;
  }, [advanced, compact, t]);

  const showSide = needsSide(kind);
  const showTrigger = needsTrigger(kind);
  const showTpSl = needsTpSl(kind);
  const showTrail = needsTrail(kind);
  const showTif = needsTif(kind);
  const showLot =
    showLotMethod &&
    (kind === 'market' ? side === 'sell' : kind !== 'limit' || side === 'sell') &&
    kind !== 'bracket';
  const keepsReason = orderKeepsReason(exchange, kind, side);
  const reasonPreview = preview as
    | (NonNullable<PaperTradeFormProps['preview']> & {
        situation?: ReasonSituation;
        situationStatus?: string;
      })
    | null
    | undefined;
  const presentReason = useMemo(
    () => presentReasonClauses(reasonPreview?.situation),
    [reasonPreview?.situation],
  );
  const selectedReason = pickedReason ?? presentReason;
  const reasonForWire = useMemo<string[] | undefined>(() => {
    if (!keepsReason) return undefined;
    if (reasonPreview?.situationStatus === 'failed' || reasonPreview?.situationStatus === 'blank') {
      return [];
    }
    if (reasonPreview?.situationStatus === 'ready') return selectedReason;
    return undefined;
  }, [keepsReason, reasonPreview?.situationStatus, selectedReason]);
  const equityVenue = exchange === 'nasdaq' || exchange === 'bist';
  const routing = bestCost && kind === 'market' && !equityVenue;
  const pacing = routing && spread;
  const minutes = spreadMinutes != null && spreadMinutes >= 1 ? Math.round(spreadMinutes) : 10;
  const slices = sliceCount != null && sliceCount >= 2 ? Math.round(sliceCount) : 10;
  const wireExchange = routing ? 'all' : exchange;
  const profit = useMemo(() => targetWire(showTpSl ? targets : []), [showTpSl, targets]);
  const canSize = canSizeByRisk(kind, side) && !routing;
  const selectedStyle: ExecutionStyleId = !routing ? 'market' : pacing ? 'spread' : 'best_cost';
  const chooseStyle = (style: ExecutionStyleId) => {
    if (style === 'market') {
      setBestCost(false);
      setSpread(false);
      return;
    }
    setBestCost(true);
    setSpread(style === 'spread');
  };
  const riskActive = canSize && riskPercent != null && riskPercent > 0;
  const stopForSize = stopLossPrice != null && stopLossPrice > 0;

  useEffect(() => {
    const next = sizedQuantity(riskPercent, preview?.sizing);
    if (next == null) return;
    setQuantity((prev) => (prev === next ? prev : next));
  }, [preview, riskPercent]);

  useEffect(() => {
    if (!onDraftChange) return;
    onDraftChange({
      product: 'spot',
      exchange: wireExchange,
      compareExchange: routing ? exchange : undefined,
      maxSlippagePct:
        kind === 'market' && !equityVenue && maxSlippagePct != null && maxSlippagePct > 0
          ? maxSlippagePct
          : undefined,
      spread: pacing || undefined,
      spreadMinutes: kind === 'market' && !equityVenue ? minutes : undefined,
      sliceCount: kind === 'market' && !equityVenue ? slices : undefined,
      symbol: symbol.trim().toUpperCase(),
      type: toApiOrderType(kind, side),
      side,
      quantity: quantity ?? 0,
      triggerPrice: showTrigger && triggerPrice != null ? triggerPrice : undefined,
      takeProfitPrice: showTpSl ? profit.takeProfitPrice : undefined,
      takeProfitTargets: showTpSl ? profit.takeProfitTargets : undefined,
      stopLossPrice: (showTpSl || riskActive) && stopLossPrice != null ? stopLossPrice : undefined,
      moveStopToEntry: showTpSl && moveStopToEntry ? true : undefined,
      riskPercent: riskActive ? (riskPercent ?? undefined) : undefined,
      trailType: showTrail ? trailType : undefined,
      trailValue: showTrail && trailValue != null ? trailValue : undefined,
      lotMethod: showLot ? lotMethod : undefined,
      reason: reasonForWire,
    });
  }, [
    onDraftChange,
    exchange,
    equityVenue,
    wireExchange,
    routing,
    pacing,
    minutes,
    slices,
    maxSlippagePct,
    symbol,
    kind,
    side,
    quantity,
    triggerPrice,
    profit,
    stopLossPrice,
    moveStopToEntry,
    trailType,
    trailValue,
    lotMethod,
    showTrigger,
    showTpSl,
    showTrail,
    showLot,
    riskActive,
    riskPercent,
    reasonForWire,
  ]);

  const submit = async (forcedSide?: 'buy' | 'sell') => {
    if (disabled || inFlightRef.current || isSubmitting) return;
    const s = forcedSide ?? side;
    if (!(quantity != null && quantity > 0)) {
      setLocalError(t('portfolio:trade.validation.quantity'));
      return;
    }
    if (showTpSl && !targetsComplete(targets)) {
      setLocalError(t('portfolio:trade.validation.targets'));
      return;
    }
    if (showTpSl && !(stopLossPrice != null && stopLossPrice > 0)) {
      setLocalError(t('portfolio:trade.validation.stopLossPrice'));
      return;
    }
    const sym = symbol.trim().toUpperCase();
    const orderType = toApiOrderType(kind, s);
    const values: PaperTradeFormValues = {
      exchange: wireExchange,
      maxSlippagePct: routing && maxSlippagePct != null && maxSlippagePct > 0 ? maxSlippagePct : undefined,
      spread: pacing || undefined,
      spreadMinutes: pacing ? minutes : undefined,
      sliceCount: pacing ? slices : undefined,
      symbol: sym,
      orderType,
      side: s,
      quantity: quantity ?? 0,
      triggerPrice: showTrigger ? (triggerPrice ?? undefined) : undefined,
      takeProfitPrice: showTpSl ? profit.takeProfitPrice : undefined,
      takeProfitTargets: showTpSl ? profit.takeProfitTargets : undefined,
      stopLossPrice: showTpSl ? (stopLossPrice ?? undefined) : undefined,
      moveStopToEntry: showTpSl && moveStopToEntry ? true : undefined,
      trailType: showTrail ? trailType : undefined,
      trailValue: showTrail ? (trailValue ?? undefined) : undefined,
      lotMethod:
        showLot &&
        (s === 'sell' || kind === 'oco' || kind === 'stop_loss' || kind === 'trailing_stop')
          ? lotMethod
          : undefined,
      timeInForce: showTif ? tif : undefined,
      reason: reasonForWire,
    };
    setLocalError(null);
    inFlightRef.current = true;
    setLocalBusy(true);
    try {
      await onSubmit(values);
      setQuantity(null);
      setRiskPercent(null);
      if (showTrigger) setTriggerPrice(null);
      if (showTpSl) {
        setTargets([{ percent: 100, offsetPercent: null, price: null }]);
        setStopLossPrice(null);
        setMoveStopToEntry(false);
      }
      if (showTrail) setTrailValue(null);
    } catch {
      // parent surfaces
    } finally {
      inFlightRef.current = false;
      setLocalBusy(false);
    }
  };

  return (
    <FormStack $compact={compact}>
      {!compact ? (
        <>
          <Text variant="h4" color="primary">
            {t('portfolio:trade.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.hint')}
          </Text>
        </>
      ) : null}

      <FieldRow>
        {!locked ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.exchange')}
              </Text>
              <Select
                value={exchange}
                aria-label={t('portfolio:trade.exchange')}
                options={['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'].map((e) => ({
                  value: e,
                  label: e,
                }))}
                onChange={setExchange}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.symbol')}
              </Text>
              <SymbolSuggest
                exchange={exchange}
                value={symbol}
                onChange={setSymbol}
                aria-label={t('portfolio:trade.symbol')}
              />
            </Field>
          </>
        ) : null}

        {kind === 'market' && !equityVenue ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.route')}
            </Text>
            <Segmented
              value={routing ? 'all' : 'venue'}
              aria-label={t('portfolio:trade.route')}
              options={[
                { label: t('portfolio:trade.thisVenue'), value: 'venue' },
                { label: t('portfolio:trade.bestCost'), value: 'all' },
              ]}
              onChange={(value) => setBestCost(value === 'all')}
            />
            {routing ? (
              <Text variant="caption" color="secondary">
                {pacing ? t('portfolio:trade.spreadHint') : t('portfolio:trade.routeHint')}
              </Text>
            ) : null}
            {routing ? (
              <Segmented
                value={pacing ? 'spread' : 'now'}
                aria-label={t('portfolio:trade.pace')}
                options={[
                  { label: t('portfolio:trade.fillNow'), value: 'now' },
                  { label: t('portfolio:trade.spread'), value: 'spread' },
                ]}
                onChange={(value) => setSpread(value === 'spread')}
              />
            ) : null}
            <InputNumber
              min={0}
              max={100}
              step={0.1}
              value={maxSlippagePct}
              onChange={(value) => setMaxSlippagePct(typeof value === 'number' ? value : null)}
              style={{ minWidth: 120 }}
              aria-label={t('portfolio:trade.maxSlippage')}
              placeholder={t('portfolio:trade.maxSlippage')}
            />
          </Field>
        ) : null}

        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.orderType')}
          </Text>
          <Select
            value={kind}
            style={{ minWidth: compact ? 120 : 140 }}
            options={kindOptions}
            onChange={(v) => setKind(v as SpotOrderKind)}
            aria-label={t('portfolio:trade.orderType')}
          />
        </Field>

        {showSide && !compact ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.side')}
            </Text>
            <Segmented
              value={side}
              options={[
                { label: t('portfolio:trade.buy'), value: 'buy' },
                { label: t('portfolio:trade.sell'), value: 'sell' },
              ]}
              onChange={(v) => setSide(v as 'buy' | 'sell')}
            />
          </Field>
        ) : null}

        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.quantity')}
          </Text>
          <InputNumber
            min={0}
            step={0.001}
            value={quantity}
            disabled={riskActive}
            onChange={(v) => {
              const next = typeof v === 'number' ? v : null;
              const sized = sizedQuantity(riskPercent, preview?.sizing);
              if (sized != null && next === sized) return;
              if (riskActive) setRiskPercent(null);
              setQuantity(next);
            }}
            style={{ minWidth: 120 }}
            aria-label={t('portfolio:trade.quantity')}
          />
        </Field>

        {canSize ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.riskPercent')}
            </Text>
            <InputNumber
              min={0}
              max={100}
              step={0.1}
              value={riskPercent}
              onChange={(v) => setRiskPercent(typeof v === 'number' ? v : null)}
              style={{ minWidth: 90 }}
              aria-label={t('portfolio:trade.riskPercent')}
            />
          </Field>
        ) : null}

        {canSize && !showTpSl ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.stopLoss')}
            </Text>
            <InputNumber
              min={0}
              step={0.01}
              value={stopLossPrice}
              onChange={(v) => setStopLossPrice(typeof v === 'number' ? v : null)}
              style={{ minWidth: 120 }}
              aria-label={t('portfolio:trade.stopLoss')}
            />
          </Field>
        ) : null}

        {showTrigger ? (
          <Field>
            <Text variant="caption" color="secondary">
              {kind === 'bracket'
                ? t('portfolio:trade.entryPrice')
                : t('portfolio:trade.triggerPrice')}
            </Text>
            <InputNumber
              min={0}
              step={0.01}
              value={triggerPrice}
              onChange={(v) => setTriggerPrice(typeof v === 'number' ? v : null)}
              style={{ minWidth: 120 }}
              aria-label={t('portfolio:trade.triggerPrice')}
            />
          </Field>
        ) : null}

        {showTpSl ? (
          <>
            <Text variant="caption" color="secondary">
              {kind === 'bracket'
                ? t('portfolio:trade.targetsHintBracket')
                : t('portfolio:trade.targetsHintOco')}
            </Text>
            <TakeProfitTargets rows={targets} onChange={setTargets} disabled={disabled || busy} />
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.stopLoss')}
              </Text>
              <InputNumber
                min={0}
                step={0.01}
                value={stopLossPrice}
                onChange={(v) => setStopLossPrice(typeof v === 'number' ? v : null)}
                style={{ minWidth: 120 }}
                aria-label={t('portfolio:trade.stopLoss')}
              />
            </Field>
            <Checkbox
              checked={moveStopToEntry}
              disabled={disabled || busy}
              data-testid="move-stop-to-entry"
              onChange={(event) => setMoveStopToEntry(event.target.checked)}
            >
              {t('portfolio:trade.moveStopToEntry')}
            </Checkbox>
          </>
        ) : null}

        {showTrail ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.trailType')}
              </Text>
              <Select
                value={trailType}
                style={{ minWidth: 110 }}
                options={[
                  { value: 'percent', label: t('portfolio:trade.trailPercent') },
                  { value: 'offset', label: t('portfolio:trade.trailOffset') },
                ]}
                onChange={setTrailType}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.trailValue')}
              </Text>
              <InputNumber
                min={0}
                step={trailType === 'percent' ? 0.01 : 0.1}
                value={trailValue}
                onChange={(v) => setTrailValue(typeof v === 'number' ? v : null)}
                style={{ minWidth: 100 }}
              />
            </Field>
          </>
        ) : null}

        {showTif ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.tif')}
            </Text>
            <Select
              value={tif}
              style={{ minWidth: 90 }}
              options={[
                { value: 'gtc', label: 'GTC' },
                { value: 'ioc', label: 'IOC' },
                { value: 'fok', label: 'FOK' },
              ]}
              onChange={setTif}
            />
          </Field>
        ) : null}

        {showLot && !compact ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.lotMethod')}
            </Text>
            <Select
              value={lotMethod}
              options={[
                { value: 'fifo', label: t('portfolio:trade.fifo') },
                { value: 'lifo', label: t('portfolio:trade.lifo') },
              ]}
              onChange={setLotMethod}
            />
          </Field>
        ) : null}
      </FieldRow>

      {riskActive ? (
        <Text variant="caption" color="secondary">
          {stopForSize
            ? t(
                kind === 'bracket'
                  ? 'portfolio:trade.riskSizesBracket'
                  : 'portfolio:trade.riskSizesBuy',
              )
            : t('portfolio:trade.riskNeedStop')}
        </Text>
      ) : null}

      {keepsReason && reasonPreview?.situationStatus === 'ready' ? (
        <>
          <Field>
            <Text variant="caption">{t('portfolio:trade.reasonTitle')}</Text>
            <Text variant="caption" color="secondary">
              {t('portfolio:trade.reasonHint')}
            </Text>
            {presentReason.map((id) => {
              const sentence =
                situationLines(
                  reasonClauseReport(reasonPreview.situation ?? {}, id),
                  t as unknown as Parameters<typeof situationLines>[1],
                )[0] ?? id;
              return (
                <Checkbox
                  key={id}
                  checked={selectedReason.includes(id)}
                  onChange={() => {
                    const base = pickedReason ?? presentReason;
                    setPickedReason(
                      base.includes(id) ? base.filter((item) => item !== id) : [...base, id],
                    );
                  }}
                >
                  {sentence}
                </Checkbox>
              );
            })}
          </Field>
          <OrderReasonHistory rows={preview?.reasonHistory} selected={selectedReason} />
        </>
      ) : null}
      {keepsReason && reasonPreview?.situationStatus === 'failed' ? (
        <Text variant="caption" color="secondary">
          {t('portfolio:trade.reasonFailed')}
        </Text>
      ) : null}
      {keepsReason && reasonPreview?.situationStatus === 'blank' ? (
        <Text variant="caption" color="secondary">
          {t('portfolio:trade.reasonBlank')}
        </Text>
      ) : null}

      {kind === 'market' && !equityVenue ? (
        <ExecutionCompare
          compare={execution}
          hasSize={quantity != null && quantity > 0}
          loading={executionLoading}
          refreshing={executionRefreshing}
          updatedAt={executionUpdatedAt}
          failed={executionFailed}
          selected={selectedStyle}
          minutes={spreadMinutes}
          slices={sliceCount}
          onSelect={chooseStyle}
          onMinutes={setSpreadMinutes}
          onSlices={setSliceCount}
        />
      ) : null}

      <OrderPreview preview={preview} loading={previewLoading} failed={previewFailed} />

      <Actions>
        {compact && (kind === 'market' || kind === 'limit') ? (
          <>
            <Button
              type="primary"
              loading={busy}
              disabled={disabled || busy}
              onPointerDown={() => setSide('buy')}
              onClick={() => void submit('buy')}
            >
              {t('portfolio:trade.submitBuy')}
            </Button>
            <Button
              danger
              loading={busy}
              disabled={disabled || busy}
              onPointerDown={() => setSide('sell')}
              onClick={() => void submit('sell')}
            >
              {t('portfolio:trade.submitSell')}
            </Button>
          </>
        ) : (
          <Button
            type="primary"
            danger={showSide && side === 'sell'}
            loading={busy}
            disabled={disabled || busy}
            onClick={() => void submit()}
          >
            {kind === 'market' || kind === 'limit'
              ? side === 'buy'
                ? t('portfolio:trade.submitBuy')
                : t('portfolio:trade.submitSell')
              : t('portfolio:trade.submit')}
          </Button>
        )}
      </Actions>

      {localError ? <Alert type="warning" showIcon message={localError} /> : null}
      {submitError ? (
        <Alert
          type="error"
          showIcon
          message={rtkErrorMessage(submitError, { resource: t('portfolio:resource') })}
        />
      ) : null}
    </FormStack>
  );
}
