import { useRef, useState } from 'react';
import { Alert, Button, Input, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { rtkErrorMessage } from '@/libs/api';
import { Actions, Field, FieldRow, Panel } from './PortfolioCashPanel.styles';
import type { PortfolioCashPanelProps } from './PortfolioCashPanel.types';

export function PortfolioCashPanel({
  isDepositing,
  isWithdrawing,
  isTransferring,
  disabled = false,
  depositError,
  withdrawError,
  transferError,
  transferBooks = [],
  onDeposit,
  onWithdraw,
  onTransfer,
}: PortfolioCashPanelProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [amount, setAmount] = useState<number | null>(null);
  const [note, setNote] = useState('');
  const [toBook, setToBook] = useState<string | undefined>(transferBooks[0]?.id);
  const [localBusy, setLocalBusy] = useState(false);
  const inFlightRef = useRef(false);

  const busy = isDepositing || isWithdrawing || isTransferring || localBusy;
  const canTransfer = Boolean(onTransfer) && transferBooks.length > 0;

  const run = async (kind: 'deposit' | 'withdraw' | 'transfer') => {
    if (disabled || amount == null || amount <= 0) return;
    if (kind === 'transfer' && (!onTransfer || !toBook)) return;
    if (isDepositing || isWithdrawing || isTransferring || inFlightRef.current) return;
    inFlightRef.current = true;
    setLocalBusy(true);
    try {
      if (kind === 'deposit') await onDeposit(amount, note || undefined);
      else if (kind === 'withdraw') await onWithdraw(amount, note || undefined);
      else if (toBook) await onTransfer?.(toBook, amount, note || undefined);
      setAmount(null);
      setNote('');
    } catch {
      // parent
    } finally {
      inFlightRef.current = false;
      setLocalBusy(false);
    }
  };

  return (
    <Panel>
      <Text variant="h4" color="primary">
        {t('portfolio:cash.title')}
      </Text>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:cash.amount')}
          </Text>
          <InputNumber
            min={0}
            value={amount}
            onChange={(v) => setAmount(typeof v === 'number' ? v : null)}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('portfolio:cash.note')}
          </Text>
          <Input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
        <Actions>
          <Button
            type="primary"
            loading={isDepositing}
            disabled={disabled || busy}
            onClick={() => void run('deposit')}
          >
            {t('portfolio:cash.deposit')}
          </Button>
          <Button
            loading={isWithdrawing}
            disabled={disabled || busy}
            onClick={() => void run('withdraw')}
          >
            {t('portfolio:cash.withdraw')}
          </Button>
        </Actions>
      </FieldRow>
      {canTransfer ? (
        <FieldRow>
          <Field>
            <Text variant="caption" color="secondary">
              {t('portfolio:cash.toBook')}
            </Text>
            <Select
              value={toBook}
              onChange={setToBook}
              options={transferBooks.map((b) => ({
                value: b.id,
                label: b.name || b.id,
              }))}
            />
          </Field>
          <Actions>
            <Button
              loading={isTransferring}
              disabled={disabled || busy || !toBook}
              onClick={() => void run('transfer')}
            >
              {t('portfolio:cash.transfer')}
            </Button>
          </Actions>
        </FieldRow>
      ) : null}
      {depositError || withdrawError || transferError ? (
        <Alert
          type="error"
          showIcon
          message={rtkErrorMessage(depositError ?? withdrawError ?? transferError, {
            resource: t('portfolio:resource'),
          })}
        />
      ) : null}
    </Panel>
  );
}
