import { useState } from 'react';
import { Alert, Button, Checkbox, InputNumber, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  rtkErrorMessage,
  type CreateScannerRuleArg,
  type ScannerCondition,
  type ScannerMatchMode,
  type ScannerRule,
} from '@/libs/api';
import { ruleConditions } from '@/libs/utils';
import { Field, FieldRow, FormStack } from './SignalsRuleForm.styles';
import type { SignalsRuleFormProps } from './SignalsRuleForm.types';

const CONDITIONS: ScannerCondition[] = ['rsi', 'ma_crossover', 'volume_increase'];

function conditionsFromRule(rule?: ScannerRule): ScannerCondition[] {
  if (!rule) {
    return ['rsi'];
  }
  const conds = ruleConditions(rule).filter((c): c is ScannerCondition =>
    CONDITIONS.includes(c as ScannerCondition),
  );
  return conds.length ? conds : ['rsi'];
}

export function SignalsRuleForm({
  intervals,
  defaultInterval = '4h',
  initialRule,
  isSubmitting = false,
  submitError,
  onSubmit,
  onCancel,
}: SignalsRuleFormProps) {
  const { t } = useTranslation(['signals', 'common']);
  const editing = initialRule != null;
  const [conditions, setConditions] = useState<ScannerCondition[]>(() => conditionsFromRule(initialRule));
  const [matchMode, setMatchMode] = useState<ScannerMatchMode>(initialRule?.matchMode ?? 'all');
  const [interval, setInterval] = useState(initialRule?.interval ?? defaultInterval);
  const [rsiPeriod, setRsiPeriod] = useState(initialRule?.rsiPeriod ?? 14);
  const [rsiCondition, setRsiCondition] = useState<'above' | 'below'>(initialRule?.rsiCondition ?? 'below');
  const [rsiThreshold, setRsiThreshold] = useState(initialRule?.rsiThreshold ?? 40);
  const [maFast, setMaFast] = useState(initialRule?.maFastPeriod ?? 12);
  const [maSlow, setMaSlow] = useState(initialRule?.maSlowPeriod ?? 26);
  const [maDir, setMaDir] = useState<'golden_cross' | 'death_cross'>(
    initialRule?.maDirection ?? 'golden_cross',
  );
  const [volLookback, setVolLookback] = useState(initialRule?.volumeLookback ?? 20);
  const [volRatio, setVolRatio] = useState(initialRule?.volumeMinRatio ?? 2);

  const intervalOptions = (intervals.length ? intervals : [interval]).map((iv) => ({
    value: iv,
    label: iv,
  }));

  const selected = new Set(conditions);
  const submit = async () => {
    if (!conditions.length) {
      return;
    }
    const body: CreateScannerRuleArg = { conditions, matchMode, interval };
    if (selected.has('rsi')) {
      body.rsiPeriod = rsiPeriod;
      body.rsiCondition = rsiCondition;
      body.rsiThreshold = rsiThreshold;
    }
    if (selected.has('ma_crossover')) {
      body.maFastPeriod = maFast;
      body.maSlowPeriod = maSlow;
      body.maDirection = maDir;
    }
    if (selected.has('volume_increase')) {
      body.volumeLookback = volLookback;
      body.volumeMinRatio = volRatio;
    }
    await onSubmit(body);
  };

  return (
    <FormStack>
      <Text variant="h4" color="primary">
        {editing ? t('signals:rules.editTitle') : t('signals:rules.createTitle')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('signals:rules.createHint')}
      </Text>
      <Field style={{ minWidth: '100%' }}>
        <Text variant="caption" color="secondary">
          {t('signals:rules.conditions')}
        </Text>
        <Checkbox.Group
          value={conditions}
          aria-label={t('signals:rules.conditions')}
          options={CONDITIONS.map((v) => ({ value: v, label: t(`signals:types.${v}`) }))}
          onChange={(vals) => setConditions(vals as ScannerCondition[])}
        />
      </Field>
      <FieldRow>
        <Field>
          <Text variant="caption" color="secondary">
            {t('signals:rules.matchMode')}
          </Text>
          <Select
            value={matchMode}
            aria-label={t('signals:rules.matchMode')}
            style={{ minWidth: 180 }}
            options={[
              { value: 'all', label: t('signals:rules.matchAll') },
              { value: 'any', label: t('signals:rules.matchAny') },
            ]}
            onChange={setMatchMode}
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
        {selected.has('rsi') ? (
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
        {selected.has('ma_crossover') ? (
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
        {selected.has('volume_increase') ? (
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
          message={editing ? t('signals:rules.updateFailed') : t('signals:rules.createFailed')}
          description={rtkErrorMessage(submitError, { resource: t('signals:resource') })}
        />
      ) : null}
      <div>
        <Button
          type="primary"
          loading={isSubmitting}
          disabled={isSubmitting || conditions.length === 0}
          onClick={() => void submit()}
        >
          {editing ? t('signals:rules.save') : t('signals:rules.create')}
        </Button>
        {editing && onCancel ? (
          <Button style={{ marginLeft: 8 }} onClick={onCancel} disabled={isSubmitting}>
            {t('common:actions.cancel')}
          </Button>
        ) : null}
      </div>
    </FormStack>
  );
}
