import { useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, InputNumber, Select, Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange } from '@/libs/api';
import { Actions, Field, FieldRow, FormStack } from './PaperTradeForm.styles';
import type { PaperTradeFormProps, PaperTradeFormValues } from './PaperTradeForm.types';
import {
  needsSide,
  needsTif,
  needsTpSl,
  needsTrail,
  needsTrigger,
  toApiOrderType,
  type SpotOrderKind,
  validateTradeForm,
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
  submitError,
  compact = false,
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
  const [takeProfitPrice, setTakeProfitPrice] = useState<number | null>(null);
  const [stopLossPrice, setStopLossPrice] = useState<number | null>(null);
  const [trailType, setTrailType] = useState<'percent' | 'offset'>('percent');
  const [trailValue, setTrailValue] = useState<number | null>(null);
  const [lotMethod, setLotMethod] = useState<'fifo' | 'lifo'>('fifo');
  const [tif, setTif] = useState<'gtc' | 'ioc' | 'fok'>('gtc');
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

  const submit = async (forcedSide?: 'buy' | 'sell') => {
    if (inFlightRef.current || isSubmitting) return;
    const s = forcedSide ?? side;
    const sym = symbol.trim().toUpperCase();
    const orderType = toApiOrderType(kind, s);
    const values: PaperTradeFormValues = {
      exchange: exchange as MarketExchange,
      symbol: sym,
      orderType,
      side: s,
      quantity: quantity ?? 0,
      triggerPrice: showTrigger ? (triggerPrice ?? undefined) : undefined,
      takeProfitPrice: showTpSl ? (takeProfitPrice ?? undefined) : undefined,
      stopLossPrice: showTpSl ? (stopLossPrice ?? undefined) : undefined,
      trailType: showTrail ? trailType : undefined,
      trailValue: showTrail ? (trailValue ?? undefined) : undefined,
      lotMethod: showLot && (s === 'sell' || kind === 'oco' || kind === 'stop_loss' || kind === 'trailing_stop')
        ? lotMethod
        : undefined,
      timeInForce: showTif ? tif : undefined,
    };
    const err = validateTradeForm(values);
    if (err) {
      setLocalError(t(`portfolio:trade.validation.${err}`, { defaultValue: t('portfolio:trade.failed') }));
      return;
    }
    setLocalError(null);
    inFlightRef.current = true;
    setLocalBusy(true);
    try {
      await onSubmit(values);
      setQuantity(null);
      if (showTrigger) setTriggerPrice(null);
      if (showTpSl) {
        setTakeProfitPrice(null);
        setStopLossPrice(null);
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
                options={['binance', 'coinbase', 'bybit'].map((e) => ({ value: e, label: e }))}
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
            onChange={(v) => setQuantity(typeof v === 'number' ? v : null)}
            style={{ minWidth: 120 }}
            aria-label={t('portfolio:trade.quantity')}
          />
        </Field>

        {showTrigger ? (
          <Field>
            <Text variant="caption" color="secondary">
              {kind === 'bracket' ? t('portfolio:trade.entryPrice') : t('portfolio:trade.triggerPrice')}
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
            <Field>
              <Text variant="caption" color="secondary">
                {t('portfolio:trade.takeProfit')}
              </Text>
              <InputNumber
                min={0}
                step={0.01}
                value={takeProfitPrice}
                onChange={(v) => setTakeProfitPrice(typeof v === 'number' ? v : null)}
                style={{ minWidth: 120 }}
              />
            </Field>
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
              />
            </Field>
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

      <Actions>
        {compact && kind === 'market' ? (
          <>
            <Button
              type="primary"
              loading={busy}
              disabled={busy}
              onClick={() => void submit('buy')}
            >
              {t('portfolio:trade.submitBuy')}
            </Button>
            <Button
              danger
              loading={busy}
              disabled={busy}
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
            disabled={busy}
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
