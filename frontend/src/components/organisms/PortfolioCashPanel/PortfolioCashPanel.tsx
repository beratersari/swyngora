import { useRef, useState } from 'react';
import { Alert, Button, Input, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { rtkErrorMessage } from '@/libs/api';
import { Actions, Field, FieldRow, Panel } from './PortfolioCashPanel.styles';
import type { PortfolioCashPanelProps } from './PortfolioCashPanel.types';

export function PortfolioCashPanel({
  isDepositing,
  isWithdrawing,
  depositError,
  withdrawError,
  onDeposit,
  onWithdraw,
}: PortfolioCashPanelProps) {
  const { t } = useTranslation(['portfolio', 'common']);
  const [amount, setAmount] = useState<number | null>(null);
  const [note, setNote] = useState('');
  const [localBusy, setLocalBusy] = useState(false);
  const inFlightRef = useRef(false);

  const busy = isDepositing || isWithdrawing || localBusy;

  const run = async (kind: 'deposit' | 'withdraw') => {
    if (amount == null || amount <= 0) return;
    if (isDepositing || isWithdrawing || inFlightRef.current) return;
    inFlightRef.current = true;
    setLocalBusy(true);
    try {
      if (kind === 'deposit') await onDeposit(amount, note || undefined);
      else await onWithdraw(amount, note || undefined);
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
            disabled={busy}
            onClick={() => void run('deposit')}
          >
            {t('portfolio:cash.deposit')}
          </Button>
          <Button
            loading={isWithdrawing}
            disabled={busy}
            onClick={() => void run('withdraw')}
          >
            {t('portfolio:cash.withdraw')}
          </Button>
        </Actions>
      </FieldRow>
      {depositError || withdrawError ? (
        <Alert
          type="error"
          showIcon
          message={rtkErrorMessage(depositError ?? withdrawError, {
            resource: t('portfolio:resource'),
          })}
        />
      ) : null}
    </Panel>
  );
}
