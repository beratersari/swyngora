import { useRef, useState } from 'react';
import { Alert, Button, InputNumber, Select, Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange, type MarginMode } from '@/libs/api';
import { Actions, Field, FieldRow, FormStack, ModeRow } from './PaperMarginForm.styles';
import type { PaperMarginFormProps, PaperMarginFormValues } from './PaperMarginForm.types';

export function PaperMarginForm({
  marginMode = 'isolated',
  modeLoading,
  isSubmitting,
  submitError,
  onModeChange,
  onSubmit,
}: PaperMarginFormProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [exchange, setExchange] = useState('binance');
  const [symbol, setSymbol] = useState('');
  const [side, setSide] = useState<'long' | 'short'>('long');
  const [orderType, setOrderType] = useState<'market' | 'limit'>('market');
  const [quantity, setQuantity] = useState<number | null>(null);
  const [leverage, setLeverage] = useState(5);
  const [limitPrice, setLimitPrice] = useState<number | null>(null);
  const [stopLoss, setStopLoss] = useState<number | null>(null);
  const [takeProfit, setTakeProfit] = useState<number | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);
  const inFlight = useRef(false);
  const [busy, setBusy] = useState(false);
  const loading = Boolean(isSubmitting || busy);

  const submit = async () => {
    if (inFlight.current || isSubmitting) return;
    const sym = symbol.trim().toUpperCase();
    if (!sym || quantity == null || quantity <= 0 || leverage < 1 || leverage > 10) {
      setLocalError(t('portfolio:margin.validation'));
      return;
    }
    if (orderType === 'limit' && !(limitPrice != null && limitPrice > 0)) {
      setLocalError(t('portfolio:margin.needLimit'));
      return;
    }
    setLocalError(null);
    const values: PaperMarginFormValues = {
      exchange: exchange as MarketExchange,
      symbol: sym,
      side,
      orderType,
      quantity,
      leverage,
      limitPrice: orderType === 'limit' ? (limitPrice ?? undefined) : undefined,
      stopLoss: stopLoss ?? undefined,
      takeProfit: takeProfit ?? undefined,
    };
    inFlight.current = true;
    setBusy(true);
    try {
      await onSubmit(values);
      setQuantity(null);
      setLimitPrice(null);
    } catch {
      /* parent */
    } finally {
      inFlight.current = false;
      setBusy(false);
    }
  };

  return (
    <FormStack>
      <Text variant="h4" color="primary">
        {t('portfolio:margin.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('portfolio:margin.hint')}
      </Text>

      <ModeRow>
        <Text variant="caption" color="secondary">
          {t('portfolio:margin.mode')}
        </Text>
        <Segmented
          value={marginMode === 'cross' ? 'cross' : 'isolated'}
          disabled={modeLoading || !onModeChange}
          options={[
            { label: t('portfolio:margin.isolated'), value: 'isolated' },
            { label: t('portfolio:margin.cross'), value: 'cross' },
          ]}
          onChange={(v) => {
            void onModeChange?.(v as MarginMode);
          }}
        />
      </ModeRow>

      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.exchange')}
          </Text>
          <Select
            value={exchange}
            style={{ minWidth: 120 }}
            options={['binance', 'coinbase', 'bybit', 'nasdaq', 'bist'].map((e) => ({ value: e, label: e }))}
            onChange={setExchange}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.symbol')}
          </Text>
          <SymbolSuggest exchange={exchange} value={symbol} onChange={setSymbol} />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:margin.side')}
          </Text>
          <Segmented
            value={side}
            options={[
              { label: t('portfolio:margin.long'), value: 'long' },
              { label: t('portfolio:margin.short'), value: 'short' },
            ]}
            onChange={(v) => setSide(v as 'long' | 'short')}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.orderType')}
          </Text>
          <Select
            value={orderType}
            style={{ minWidth: 110 }}
            options={[
              { value: 'market', label: t('portfolio:trade.kinds.market') },
              { value: 'limit', label: t('portfolio:trade.kinds.limit') },
            ]}
            onChange={setOrderType}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.quantity')}
          </Text>
          <InputNumber
            min={0}
            step={0.001}
            value={quantity}
            onChange={(v) => setQuantity(typeof v === 'number' ? v : null)}
            style={{ minWidth: 110 }}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:margin.leverage')}
          </Text>
          <InputNumber
            min={1}
            max={10}
            step={1}
            value={leverage}
            onChange={(v) => setLeverage(typeof v === 'number' ? v : 1)}
            style={{ minWidth: 80 }}
          />
        </Field>
        {orderType === 'limit' ? (
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:margin.limitPrice')}
            </Text>
            <InputNumber
              min={0}
              step={0.01}
              value={limitPrice}
              onChange={(v) => setLimitPrice(typeof v === 'number' ? v : null)}
              style={{ minWidth: 110 }}
            />
          </Field>
        ) : null}
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.stopLoss')}
          </Text>
          <InputNumber
            min={0}
            step={0.01}
            value={stopLoss}
            onChange={(v) => setStopLoss(typeof v === 'number' ? v : null)}
            style={{ minWidth: 100 }}
            placeholder="—"
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:trade.takeProfit')}
          </Text>
          <InputNumber
            min={0}
            step={0.01}
            value={takeProfit}
            onChange={(v) => setTakeProfit(typeof v === 'number' ? v : null)}
            style={{ minWidth: 100 }}
            placeholder="—"
          />
        </Field>
      </FieldRow>

      <Actions>
        <Button type="primary" loading={loading} disabled={loading} onClick={() => void submit()}>
          {t('portfolio:margin.submit')}
        </Button>
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
