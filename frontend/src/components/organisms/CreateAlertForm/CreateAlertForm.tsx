import { useEffect, useState } from 'react';
import { Alert, Button, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange } from '@/libs/api';
import { Field, FieldRow, FormStack } from './CreateAlertForm.styles';
import type { CreateAlertFormProps } from './CreateAlertForm.types';

export type AlertKind = 'price' | 'liquidation_feed' | 'liquidation_cascade';

export type CreatePriceAlertValues = {
  exchange: string;
  symbol?: string;
  kind?: AlertKind;
  condition?: string;
  targetPrice?: number;
  mode: 'one_time' | 'repeating';
};

/**
 * Presentational create form for develop PriceAlert API.
 * Page owns RTK mutation.
 */
export function CreateAlertForm({
  defaultExchange = 'binance',
  defaultSymbol = '',
  isSubmitting = false,
  submitError,
  onSubmit,
}: CreateAlertFormProps & {
  isSubmitting?: boolean;
  submitError?: unknown;
  onSubmit: (values: CreatePriceAlertValues) => Promise<void>;
}) {
  const { t } = useTranslation(['alerts', 'common']);
  const [kind, setKind] = useState<AlertKind>('price');
  const [exchange, setExchange] = useState(String(defaultExchange));
  const [symbol, setSymbol] = useState(defaultSymbol);
  const [condition, setCondition] = useState('above');
  const [targetPrice, setTargetPrice] = useState<number | null>(null);
  const [mode, setMode] = useState<'one_time' | 'repeating'>('one_time');
  const isFeed = kind === 'liquidation_feed';
  const isCascade = kind === 'liquidation_cascade';
  const liqExchanges = ['binance', 'bybit', 'all'];
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
    if (isCascade && !symbol.trim()) {
      return;
    }
    try {
      await onSubmit({
        kind,
        exchange: isFeed || isCascade ? exchange || 'all' : (exchange as MarketExchange),
        symbol: isFeed ? 'ALL' : symbol.trim(),
        condition: isFeed ? 'down' : isCascade ? condition || 'cascade' : condition,
        targetPrice: isFeed ? targetPrice ?? 300 : isCascade ? 0 : (targetPrice ?? 0),
        mode: isFeed || isCascade ? 'repeating' : mode,
      });
      if (kind === 'price') setTargetPrice(null);
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
            ]}
            onChange={(v) => {
              setKind(v);
              if (v === 'liquidation_feed' || v === 'liquidation_cascade') {
                setMode('repeating');
                if (v === 'liquidation_cascade') setCondition('cascade');
                if (!['binance', 'bybit', 'all'].includes(exchange)) setExchange('all');
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
            options={(isFeed || isCascade ? liqExchanges : spotExchanges).map((e) => ({ value: e, label: e }))}
            onChange={setExchange}
          />
        </Field>
        {isFeed ? null : (
          <Field>
            <Text variant="caption" color="secondary">
              {t('alerts:symbol')}
            </Text>
            <SymbolSuggest
              exchange={exchange === 'all' ? 'binance' : exchange}
              value={symbol}
              onChange={setSymbol}
              aria-label={t('alerts:symbol')}
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
                  : [
                      { value: 'above', label: t('alerts:conditions.above', { defaultValue: 'Price above' }) },
                      { value: 'below', label: t('alerts:conditions.below', { defaultValue: 'Price below' }) },
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
                : t('alerts:threshold', { defaultValue: 'Target price' })}
            </Text>
            <InputNumber
              value={isFeed ? (targetPrice ?? 300) : (targetPrice ?? undefined)}
              aria-label={isFeed ? t('alerts:minDownSeconds', { defaultValue: 'Down for (seconds)' }) : t('alerts:threshold', { defaultValue: 'Target price' })}
              min={isFeed ? 30 : 0}
              style={{ minWidth: 140 }}
              onChange={(v) => setTargetPrice(typeof v === 'number' ? v : null)}
            />
          </Field>
        )}
        {isFeed || isCascade ? null : (
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
            (isCascade && !symbol.trim())
          }
          onClick={() => void submit()}
        >
          {t('alerts:create')}
        </Button>
      </div>
    </FormStack>
  );
}
