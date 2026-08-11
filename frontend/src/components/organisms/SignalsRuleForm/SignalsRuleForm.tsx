import { useState } from 'react';
import { Alert, Button, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { rtkErrorMessage, type CreateScannerRuleArg, type ScannerRuleType } from '@/libs/api';
import { Field, FieldRow, FormStack } from './SignalsRuleForm.styles';
import type { SignalsRuleFormProps } from './SignalsRuleForm.types';

const TYPES: ScannerRuleType[] = ['rsi', 'ma_crossover', 'volume_increase'];

export function SignalsRuleForm({
  intervals,
  defaultInterval = '4h',
  isSubmitting = false,
  submitError,
  onSubmit,
}: SignalsRuleFormProps) {
  const { t } = useTranslation(['signals', 'common']);
  const [type, setType] = useState<ScannerRuleType>('rsi');
  const [interval, setInterval] = useState(defaultInterval);
  const [rsiPeriod, setRsiPeriod] = useState(14);
  const [rsiCondition, setRsiCondition] = useState<'above' | 'below'>('below');
  const [rsiThreshold, setRsiThreshold] = useState(40);
  const [maFast, setMaFast] = useState(12);
  const [maSlow, setMaSlow] = useState(26);
  const [maDir, setMaDir] = useState<'golden_cross' | 'death_cross'>('golden_cross');
  const [volLookback, setVolLookback] = useState(20);
  const [volRatio, setVolRatio] = useState(2);

  const intervalOptions = (intervals.length ? intervals : [interval]).map((iv) => ({
    value: iv,
    label: iv,
  }));

  const submit = async () => {
    const body: CreateScannerRuleArg = { type, interval };
    if (type === 'rsi') {
      body.rsiPeriod = rsiPeriod;
      body.rsiCondition = rsiCondition;
      body.rsiThreshold = rsiThreshold;
    } else if (type === 'ma_crossover') {
      body.maFastPeriod = maFast;
      body.maSlowPeriod = maSlow;
      body.maDirection = maDir;
    } else {
      body.volumeLookback = volLookback;
      body.volumeMinRatio = volRatio;
    }
    await onSubmit(body);
  };

  return (
    <FormStack>
      <Text variant="h4" color="primary">
        {t('signals:rules.createTitle')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('signals:rules.createHint')}
      </Text>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:rules.type')}
          </Text>
          <Select
            value={type}
            aria-label={t('signals:rules.type')}
            style={{ minWidth: 180 }}
            options={TYPES.map((v) => ({ value: v, label: t(`signals:types.${v}`) }))}
            onChange={setType}
          />
        </Field>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:interval')}
          </Text>
          <Select
            value={interval}
            aria-label={t('signals:interval')}
            style={{ minWidth: 100 }}
            options={intervalOptions}
            onChange={setInterval}
            showSearch
          />
        </Field>
        {type === 'rsi' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.rsiPeriod')}
              </Text>
              <InputNumber
                min={2}
                max={200}
                value={rsiPeriod}
                aria-label={t('signals:rules.rsiPeriod')}
                onChange={(v) => setRsiPeriod(typeof v === 'number' ? v : 14)}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.rsiCondition')}
              </Text>
              <Select
                value={rsiCondition}
                aria-label={t('signals:rules.rsiCondition')}
                style={{ minWidth: 120 }}
                options={[
                  { value: 'below', label: t('signals:rules.below') },
                  { value: 'above', label: t('signals:rules.above') },
                ]}
                onChange={setRsiCondition}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.rsiThreshold')}
              </Text>
              <InputNumber
                min={0}
                max={100}
                value={rsiThreshold}
                aria-label={t('signals:rules.rsiThreshold')}
                onChange={(v) => setRsiThreshold(typeof v === 'number' ? v : 40)}
              />
            </Field>
          </>
        ) : null}
        {type === 'ma_crossover' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.maFast')}
              </Text>
              <InputNumber
                min={2}
                max={200}
                value={maFast}
                aria-label={t('signals:rules.maFast')}
                onChange={(v) => setMaFast(typeof v === 'number' ? v : 12)}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.maSlow')}
              </Text>
              <InputNumber
                min={3}
                max={400}
                value={maSlow}
                aria-label={t('signals:rules.maSlow')}
                onChange={(v) => setMaSlow(typeof v === 'number' ? v : 26)}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.maDirection')}
              </Text>
              <Select
                value={maDir}
                aria-label={t('signals:rules.maDirection')}
                style={{ minWidth: 150 }}
                options={[
                  { value: 'golden_cross', label: t('signals:rules.goldenCross') },
                  { value: 'death_cross', label: t('signals:rules.deathCross') },
                ]}
                onChange={setMaDir}
              />
            </Field>
          </>
        ) : null}
        {type === 'volume_increase' ? (
          <>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.volLookback')}
              </Text>
              <InputNumber
                min={2}
                max={200}
                value={volLookback}
                aria-label={t('signals:rules.volLookback')}
                onChange={(v) => setVolLookback(typeof v === 'number' ? v : 20)}
              />
            </Field>
            <Field>
              <Text variant="caption" color="secondary">
                {t('signals:rules.volRatio')}
              </Text>
              <InputNumber
                min={1.01}
                max={100}
                step={0.1}
                value={volRatio}
                aria-label={t('signals:rules.volRatio')}
                onChange={(v) => setVolRatio(typeof v === 'number' ? v : 2)}
              />
            </Field>
          </>
        ) : null}
      </FieldRow>
      {submitError != null ? (
        <Alert
          type="error"
          showIcon
          message={t('signals:rules.createFailed')}
          description={rtkErrorMessage(submitError, { resource: t('signals:resource') })}
        />
      ) : null}
      <div>
        <Button type="primary" loading={isSubmitting} disabled={isSubmitting} onClick={() => void submit()}>
          {t('signals:rules.create')}
        </Button>
      </div>
    </FormStack>
  );
}
