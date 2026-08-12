import { useEffect, useRef, useState } from 'react';
import { Alert, Button, InputNumber, Select, Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { SymbolSuggest } from '@/components/molecules/SymbolSuggest';
import { rtkErrorMessage, type MarketExchange } from '@/libs/api';
import { Actions, Field, FieldRow, FormStack } from './PaperTradeForm.styles';
import type { PaperTradeFormProps, PaperTradeFormValues } from './PaperTradeForm.types';

/**
 * Presentational paper market order form. Page owns RTK mutation.
 */
export function PaperTradeForm({
  lockedExchange,
  lockedSymbol,
  defaultExchange = 'binance',
  defaultSymbol = '',
  defaultSide = 'buy',
  showLotMethod = true,
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
  const [quantity, setQuantity] = useState<number | null>(null);
  const [lotMethod, setLotMethod] = useState<'fifo' | 'lifo'>('fifo');
  const [localBusy, setLocalBusy] = useState(false);
  const inFlightRef = useRef(false);

  useEffect(() => {
    if (lockedExchange) setExchange(String(lockedExchange));
  }, [lockedExchange]);
  useEffect(() => {
    if (lockedSymbol) setSymbol(lockedSymbol);
  }, [lockedSymbol]);

  const busy = isSubmitting || localBusy;

  const submit = async (nextSide?: 'buy' | 'sell') => {
    if (isSubmitting || inFlightRef.current) return;
    const s = nextSide ?? side;
    const sym = symbol.trim().toUpperCase();
    if (!sym || quantity == null || !Number.isFinite(quantity) || quantity <= 0) return;
    const values: PaperTradeFormValues = {
      exchange: exchange as MarketExchange,
      symbol: sym,
      side: s,
      quantity,
      lotMethod: s === 'sell' ? lotMethod : undefined,
    };
    inFlightRef.current = true;
    setLocalBusy(true);
    try {
      await onSubmit(values);
      setQuantity(null);
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
        {!compact ? (
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
            aria-label={t('portfolio:trade.quantity')}
          />
        </Field>
        {showLotMethod && side === 'sell' && !compact ? (
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
        {compact ? (
          <>
            <Button
              type="primary"
              loading={isSubmitting}
              disabled={busy}
              onClick={() => void submit('buy')}
            >
              {t('portfolio:trade.submitBuy')}
            </Button>
            <Button
              danger
              loading={isSubmitting}
              disabled={busy}
              onClick={() => void submit('sell')}
            >
              {t('portfolio:trade.submitSell')}
            </Button>
          </>
        ) : (
          <Button
            type="primary"
            danger={side === 'sell'}
            loading={isSubmitting}
            disabled={busy}
            onClick={() => void submit()}
          >
            {side === 'buy' ? t('portfolio:trade.submitBuy') : t('portfolio:trade.submitSell')}
          </Button>
        )}
      </Actions>

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
