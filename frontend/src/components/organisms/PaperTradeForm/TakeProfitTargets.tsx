import { Button, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { Field, FieldRow } from './PaperTradeForm.styles';
import {
  MAX_TAKE_PROFIT_TARGETS,
  emptyTargetRow,
  exampleScaleOut,
  targetShareSum,
  type TargetRow,
} from './targets';

type TakeProfitTargetsProps = {
  rows: TargetRow[];
  onChange: (rows: TargetRow[]) => void;
  disabled?: boolean;
};

/**
 * Scale-out rows for an OCO or bracket. The page sends them with the rest of the ticket.
 */
export function TakeProfitTargets({ rows, onChange, disabled = false }: TakeProfitTargetsProps) {
  const { t } = useTranslation('portfolio');
  const sum = targetShareSum(rows);
  const sumOk = Math.abs(sum - 100) <= 0.01;

  const update = (index: number, patch: Partial<TargetRow>) => {
    onChange(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };

  return (
    <div style={{ flexBasis: '100%', display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Text variant="caption">{t('trade.targetsTitle')}</Text>
      {rows.map((row, index) => {
        const gainSet = row.offsetPercent != null && row.offsetPercent > 0;
        return (
          <FieldRow key={index}>
            <Field>
              <Text variant="caption" color="secondary">
                {t('trade.targetShare')}
              </Text>
              <InputNumber
                min={0}
                max={100}
                step={5}
                value={row.percent}
                disabled={disabled}
                onChange={(value) => update(index, { percent: typeof value === 'number' ? value : null })}
                aria-label={t('trade.targetShare')}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('trade.targetGain')}
              </Text>
              <InputNumber
                min={0}
                step={1}
                value={row.offsetPercent}
                disabled={disabled}
                onChange={(value) =>
                  update(index, {
                    offsetPercent: typeof value === 'number' ? value : null,
                    price: typeof value === 'number' && value > 0 ? null : row.price,
                  })
                }
                aria-label={t('trade.targetGain')}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('trade.takeProfit')}
              </Text>
              <InputNumber
                min={0}
                step={0.01}
                value={row.price}
                disabled={disabled || gainSet}
                onChange={(value) => update(index, { price: typeof value === 'number' ? value : null })}
                aria-label={t('trade.takeProfit')}
              />
            </Field>
            {rows.length > 1 ? (
              <Button
                type="text"
                disabled={disabled}
                onClick={() => onChange(rows.filter((_, i) => i !== index))}
                aria-label={t('trade.removeTarget')}
              >
                {t('trade.removeTarget')}
              </Button>
            ) : null}
          </FieldRow>
        );
      })}
      <FieldRow>
        <Button
          disabled={disabled || rows.length >= MAX_TAKE_PROFIT_TARGETS}
          onClick={() => onChange([...rows, emptyTargetRow()])}
        >
          {t('trade.addTarget')}
        </Button>
        <Button disabled={disabled} onClick={() => onChange(exampleScaleOut())}>
          {t('trade.scaleOutExample')}
        </Button>
      </FieldRow>
      <Text variant="caption" color={sumOk ? 'secondary' : 'error'}>
        {sumOk ? t('trade.targetsSumOk') : t('trade.targetsSum', { sum })}
      </Text>
    </div>
  );
}
