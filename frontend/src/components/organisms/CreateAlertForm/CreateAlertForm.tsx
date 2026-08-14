import { useEffect, useState } from 'react';
import { Alert, Button, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange } from '@/libs/api';
import { Field, FieldRow, FormStack } from './CreateAlertForm.styles';
import type { CreateAlertFormProps } from './CreateAlertForm.types';

export type CreatePriceAlertValues = {
  exchange: MarketExchange;
  symbol: string;
  condition: 'above' | 'below';
  targetPrice: number;
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
  const [exchange, setExchange] = useState(String(defaultExchange));
  const [symbol, setSymbol] = useState(defaultSymbol);
  const [condition, setCondition] = useState<'above' | 'below'>('above');
  const [targetPrice, setTargetPrice] = useState<number | null>(null);
  const [mode, setMode] = useState<'one_time' | 'repeating'>('one_time');

  useEffect(() => {
    setExchange(String(defaultExchange || 'binance'));
  }, [defaultExchange]);
  useEffect(() => {
    setSymbol(defaultSymbol || '');
  }, [defaultSymbol]);

  const submit = async () => {
    if (!symbol.trim() || targetPrice == null || !Number.isFinite(targetPrice) || targetPrice <= 0) {
      return;
    }
    try {
      await onSubmit({
        exchange: exchange as MarketExchange,
        symbol: symbol.trim(),
        condition,
        targetPrice,
        mode,
      });
      setTargetPrice(null);
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
            {t('alerts:exchange')}
          </Text>
          <Select
            value={exchange}
            aria-label={t('alerts:exchange')}
            style={{ minWidth: 120 }}
            options={['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'].map((e) => ({ value: e, label: e }))}
            onChange={setExchange}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('alerts:symbol')}
          </Text>
          <SymbolSuggest
            exchange={exchange}
            value={symbol}
            onChange={setSymbol}
            aria-label={t('alerts:symbol')}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('alerts:condition')}
          </Text>
          <Select
            value={condition}
            aria-label={t('alerts:condition')}
            style={{ minWidth: 120 }}
            options={[
              { value: 'above', label: t('alerts:conditions.above', { defaultValue: 'Price above' }) },
              { value: 'below', label: t('alerts:conditions.below', { defaultValue: 'Price below' }) },
            ]}
            onChange={(v) => setCondition(v)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('alerts:threshold', { defaultValue: 'Target price' })}
          </Text>
          <InputNumber
            value={targetPrice ?? undefined}
            aria-label={t('alerts:threshold', { defaultValue: 'Target price' })}
            min={0}
            style={{ minWidth: 120 }}
            onChange={(v) => setTargetPrice(typeof v === 'number' ? v : null)}
          />
        </Field>
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
          disabled={!symbol.trim() || targetPrice == null || isSubmitting}
          onClick={() => void submit()}
        >
          {t('alerts:create')}
        </Button>
      </div>
    </FormStack>
  );
}
