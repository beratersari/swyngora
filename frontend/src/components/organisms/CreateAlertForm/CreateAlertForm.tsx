import { useEffect, useState } from 'react';
import { Alert, Button, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange } from '@/libs/api';
import { Field, FieldRow, FormStack } from './CreateAlertForm.styles';
import type { AlertKind, CreateAlertFormProps } from './CreateAlertForm.types';

export type { AlertKind } from './CreateAlertForm.types';

export type CreatePriceAlertValues = {
  exchange: string;
  symbol?: string;
  kind?: AlertKind;
  condition?: string;
  targetPrice?: number;
  window?: string;
  mode: 'one_time' | 'repeating';
};

/**
 * Presentational create form for develop PriceAlert API.
 * Page owns RTK mutation.
 */
export function CreateAlertForm({
  defaultExchange = 'binance',
  defaultSymbol = '',
  defaultKind = 'price',
  isSubmitting = false,
  submitError,
  onSubmit,
}: CreateAlertFormProps & {
  isSubmitting?: boolean;
  submitError?: unknown;
  onSubmit: (values: CreatePriceAlertValues) => Promise<void>;
}) {
  const { t } = useTranslation(['alerts', 'common']);
  const [kind, setKind] = useState<AlertKind>(defaultKind);
  const [exchange, setExchange] = useState(String(defaultExchange));
  const [symbol, setSymbol] = useState(defaultSymbol);
  const [condition, setCondition] = useState(
    defaultKind === 'setup_radar' || defaultKind === 'liquidation_level'
      ? 'any'
      : defaultKind === 'liquidation_cascade'
        ? 'cascade'
        : defaultKind === 'liquidation_notional'
          ? 'both'
          : 'above',
  );
  const [targetPrice, setTargetPrice] = useState<number | null>(defaultKind === 'setup_radar' ? 60 : null);
  const [window, setWindow] = useState('5m');
  const [mode, setMode] = useState<'one_time' | 'repeating'>(
    defaultKind === 'price' ? 'one_time' : 'repeating',
  );
  const isFeed = kind === 'liquidation_feed';
  const isCascade = kind === 'liquidation_cascade';
  const isNotional = kind === 'liquidation_notional';
  const isLevel = kind === 'liquidation_level';
  const isRadar = kind === 'setup_radar';
  const isFunding = kind === 'funding';
  const isOI = kind === 'open_interest';
  const isTaker = kind === 'taker';
  const isFear = kind === 'fear_greed';
  const isTape = isFunding || isOI || isTaker || isFear;
  const isLiq = isFeed || isCascade || isNotional || isLevel;
  const venueWide = isLiq || isRadar || isTape;
  const lockRepeating = isLiq || isRadar;
  const liqExchanges = ['binance', 'bybit', 'okx', 'all'];
  const spotExchanges = ['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'];

  useEffect(() => {
    setExchange(String(defaultExchange || 'binance'));
  }, [defaultExchange]);
  useEffect(() => {
    setSymbol(defaultSymbol || '');
  }, [defaultSymbol]);

  const submit = async () => {
    if (kind === 'price') {
      if (!symbol.trim() || targetPrice == null || !Number.isFinite(targetPrice) || targetPrice <= 0) {
        return;
      }
    }
    if ((isCascade || isNotional || isLevel) && !symbol.trim()) {
      return;
    }
    if (isNotional && (targetPrice == null || targetPrice <= 0)) {
      return;
    }
    try {
      await onSubmit({
        kind,
        exchange: venueWide ? exchange || 'all' : (exchange as MarketExchange),
        symbol: isFeed || isFear ? 'ALL' : isRadar ? symbol.trim() || 'ALL' : symbol.trim(),
        condition: isFeed
          ? 'down'
          : isCascade
            ? condition || 'cascade'
            : isNotional
              ? condition || 'both'
              : isLevel
                ? condition || 'any'
                : isRadar
                  ? condition || 'any'
                  : condition,
        targetPrice: isFeed
          ? targetPrice ?? 300
          : isCascade
            ? 0
            : isLevel
              ? targetPrice ?? 0
              : isRadar
                ? targetPrice ?? 60
                : (targetPrice ?? 0),
        window: isNotional ? window : undefined,
        mode: lockRepeating ? 'repeating' : mode,
      });
      if (kind === 'price' || isNotional) setTargetPrice(null);
    } catch {
      // parent surfaces error
    }
  };

  return (
    <FormStack>
      <Text variant="h4" color="primary">
        {t('alerts:createTitle')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('alerts:createHint')}
      </Text>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('alerts:kind', { defaultValue: 'Kind' })}
          </Text>
          <Select
            value={kind}
            aria-label={t('alerts:kind', { defaultValue: 'Kind' })}
            style={{ minWidth: 180 }}
            options={[
              { value: 'price', label: t('alerts:kinds.price', { defaultValue: 'Price' }) },
              { value: 'liquidation_feed', label: t('alerts:kinds.liquidation_feed', { defaultValue: 'Liquidation feed' }) },
              { value: 'liquidation_cascade', label: t('alerts:kinds.liquidation_cascade', { defaultValue: 'Liquidation cascade' }) },
              { value: 'liquidation_notional', label: t('alerts:kinds.liquidation_notional', { defaultValue: 'Liquidation notional' }) },
              { value: 'liquidation_level', label: t('alerts:kinds.liquidation_level', { defaultValue: 'Liquidation level' }) },
              { value: 'setup_radar', label: t('alerts:kinds.setup_radar', { defaultValue: 'Setup Radar' }) },
              { value: 'funding', label: t('alerts:kinds.funding', { defaultValue: 'Funding' }) },
              { value: 'open_interest', label: t('alerts:kinds.open_interest', { defaultValue: 'Open interest' }) },
              { value: 'taker', label: t('alerts:kinds.taker', { defaultValue: 'Taker buy share' }) },
              { value: 'fear_greed', label: t('alerts:kinds.fear_greed', { defaultValue: 'Fear and greed' }) },
            ]}
            onChange={(v) => {
              setKind(v);
              if (v === 'funding' || v === 'open_interest' || v === 'taker' || v === 'fear_greed') {
                setCondition('above');
                setMode('one_time');
                setTargetPrice(null);
                if (!['binance', 'bybit', 'okx', 'all'].includes(exchange)) setExchange('all');
              } else if (v === 'liquidation_feed' || v === 'liquidation_cascade' || v === 'liquidation_notional' || v === 'liquidation_level' || v === 'setup_radar') {
                setMode('repeating');
                if (v === 'liquidation_cascade') setCondition('cascade');
                if (v === 'liquidation_notional') setCondition('both');
                if (v === 'liquidation_level') {
                  setCondition('any');
                  setTargetPrice(null);
                }
                if (v === 'setup_radar') {
                  setCondition('any');
                  setTargetPrice(60);
                }
                if (!['binance', 'bybit', 'okx', 'all'].includes(exchange)) setExchange('all');
              } else {
                setCondition('above');
                if (exchange === 'all') setExchange('binance');
              }
            }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('alerts:exchange')}
          </Text>
          <Select
            value={exchange}
            aria-label={t('alerts:exchange')}
            style={{ minWidth: 120 }}
            options={(venueWide ? liqExchanges : spotExchanges).map((e) => ({ value: e, label: e }))}
            onChange={setExchange}
          />
        </Field>
        {isFeed || isFear ? null : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('alerts:symbol')}
            </Text>
            <SymbolSuggest
              exchange={exchange === 'all' ? 'binance' : exchange}
              value={symbol}
              onChange={setSymbol}
              aria-label={t('alerts:symbol')}
              placeholder={isRadar ? t('alerts:symbolOrAll', { defaultValue: 'BTCUSDT or empty for all' }) : undefined}
            />
          </Field>
        )}
        {isFeed ? null : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('alerts:condition')}
            </Text>
            <Select
              value={condition}
              aria-label={t('alerts:condition')}
              style={{ minWidth: 140 }}
              options={
                isCascade
                  ? [
                      { value: 'cascade', label: t('alerts:conditions.cascade', { defaultValue: 'Cascade' }) },
                      { value: 'extreme', label: t('alerts:conditions.extreme', { defaultValue: 'Extreme' }) },
                    ]
                  : isNotional
                    ? [
                        { value: 'both', label: t('alerts:conditions.both', { defaultValue: 'Long + short' }) },
                        { value: 'long', label: t('alerts:conditions.long', { defaultValue: 'Longs' }) },
                        { value: 'short', label: t('alerts:conditions.short', { defaultValue: 'Shorts' }) },
                      ]
                    : isLevel
                      ? [
                          { value: 'any', label: t('alerts:conditions.levelAny', { defaultValue: 'Any proven-enough band' }) },
                          { value: 'fair', label: t('alerts:conditions.levelFair', { defaultValue: 'Fair or better' }) },
                          { value: 'high', label: t('alerts:conditions.levelHigh', { defaultValue: 'High confidence only' }) },
                        ]
                    : isRadar
                      ? [
                          { value: 'any', label: t('alerts:conditions.any', { defaultValue: 'Any after-path' }) },
                          { value: 'up', label: t('alerts:conditions.up', { defaultValue: 'Usually rose after' }) },
                          { value: 'down', label: t('alerts:conditions.down', { defaultValue: 'Usually fell after' }) },
                        ]
                      : [
                          { value: 'above', label: t(isTape ? 'alerts:conditions.levelAbove' : 'alerts:conditions.above', { defaultValue: isTape ? 'Above' : 'Price above' }) },
                          { value: 'below', label: t(isTape ? 'alerts:conditions.levelBelow' : 'alerts:conditions.below', { defaultValue: isTape ? 'Below' : 'Price below' }) },
                        ]
              }
              onChange={(v) => setCondition(v)}
            />
          </Field>
        )}
        {isCascade ? null : (
          <Field>
            <Text variant="caption" color="secondary">
              {isFeed
                ? t('alerts:minDownSeconds', { defaultValue: 'Down for (seconds)' })
                : isNotional
                  ? t('alerts:notional', { defaultValue: 'USDT notional' })
                  : isLevel
                    ? t('alerts:minConfidence', { defaultValue: 'Min confidence (0–1)' })
                  : isRadar
                    ? t('alerts:minSimilarity', { defaultValue: 'Min similarity' })
                    : isFunding
                      ? t('alerts:fundingLevel', { defaultValue: 'Funding %' })
                      : isOI
                        ? t('alerts:oiLevel', { defaultValue: 'Open interest USDT' })
                        : isTaker
                          ? t('alerts:takerLevel', { defaultValue: '5m buy share %' })
                          : isFear
                            ? t('alerts:fearLevel', { defaultValue: 'Score 0–100' })
                            : t('alerts:threshold', { defaultValue: 'Target price' })}
            </Text>
            <InputNumber
              value={isFeed ? (targetPrice ?? 300) : isRadar ? (targetPrice ?? 60) : isLevel ? (targetPrice ?? 0) : (targetPrice ?? undefined)}
              aria-label={
                isFeed
                  ? t('alerts:minDownSeconds', { defaultValue: 'Down for (seconds)' })
                  : isNotional
                    ? t('alerts:notional', { defaultValue: 'USDT notional' })
                    : isLevel
                      ? t('alerts:minConfidence', { defaultValue: 'Min confidence (0–1)' })
                      : isRadar
                        ? t('alerts:minSimilarity', { defaultValue: 'Min similarity' })
                        : isFunding
                          ? t('alerts:fundingLevel', { defaultValue: 'Funding %' })
                          : isOI
                            ? t('alerts:oiLevel', { defaultValue: 'Open interest USDT' })
                            : isTaker
                              ? t('alerts:takerLevel', { defaultValue: '5m buy share %' })
                              : isFear
                                ? t('alerts:fearLevel', { defaultValue: 'Score 0–100' })
                                : t('alerts:threshold', { defaultValue: 'Target price' })
              }
              min={isFeed ? 30 : isRadar ? 40 : isFunding ? -5 : 0}
              max={isRadar || isTaker || isFear ? 100 : isLevel ? 1 : isFunding ? 5 : undefined}
              step={isLevel || isFunding ? 0.01 : isTaker || isFear ? 1 : undefined}
              style={{ minWidth: 160 }}
              onChange={(v) => setTargetPrice(typeof v === 'number' ? v : null)}
            />
          </Field>
        )}
        {isNotional ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('alerts:window', { defaultValue: 'Window' })}
            </Text>
            <Select
              value={window}
              aria-label={t('alerts:window', { defaultValue: 'Window' })}
              style={{ minWidth: 100 }}
              options={[
                { value: '1m', label: '1m' },
                { value: '5m', label: '5m' },
                { value: '15m', label: '15m' },
                { value: '1h', label: '1h' },
              ]}
              onChange={setWindow}
            />
          </Field>
        ) : null}
        {lockRepeating ? null : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('alerts:mode', { defaultValue: 'Mode' })}
            </Text>
            <Select
              value={mode}
              aria-label={t('alerts:mode', { defaultValue: 'Mode' })}
              style={{ minWidth: 140 }}
              options={[
                { value: 'one_time', label: t('alerts:modes.one_time', { defaultValue: 'One-time' }) },
                { value: 'repeating', label: t('alerts:modes.repeating', { defaultValue: 'Repeating' }) },
              ]}
              onChange={(v) => setMode(v)}
            />
          </Field>
        )}
      </FieldRow>
      {submitError != null ? (
        <Alert
          type="error"
          showIcon
          message={t('alerts:createFailed')}
          description={rtkErrorMessage(submitError, { resource: t('alerts:resource') })}
        />
      ) : null}
      <div>
        <Button
          type="primary"
          loading={isSubmitting}
          disabled={
            isSubmitting ||
            (kind === 'price' && (!symbol.trim() || targetPrice == null)) ||
            (isCascade && !symbol.trim()) ||
            (isLevel && !symbol.trim()) ||
            (isNotional && (!symbol.trim() || targetPrice == null || targetPrice <= 0)) ||
            (isRadar && targetPrice != null && (targetPrice < 40 || targetPrice > 100)) ||
            (isTape && targetPrice == null) ||
            ((isFunding || isOI || isTaker) && !symbol.trim())
          }
          onClick={() => void submit()}
        >
          {t('alerts:create')}
        </Button>
      </div>
    </FormStack>
  );
}
