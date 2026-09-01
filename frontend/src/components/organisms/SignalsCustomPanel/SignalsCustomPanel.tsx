import { useMemo, useState } from 'react';
import { Alert, Button, Input, InputNumber, Select, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { rtkErrorMessage } from '@/libs/api';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import { CUSTOM_SIGNAL_BUILT_IN_ID, CUSTOM_SIGNAL_FACTORS } from './constants';
import { customWeightTotal, customWeightsValid, draftNeedsHunt, previewCustomMix } from './helpers';
import {
  ActionRow,
  Card,
  CardTop,
  FactorBreakdown,
  FactorLine,
  FactorList,
  FactorRow,
  Layout,
  Mixer,
  Results,
} from './SignalsCustomPanel.styles';
import type { SignalsCustomPanelProps } from './SignalsCustomPanel.types';

function gradeVariant(grade?: string): 'gradeA' | 'gradeB' | 'gradeC' {
  if (grade === 'A') return 'gradeA';
  if (grade === 'B') return 'gradeB';
  return 'gradeC';
}

/**
 * Custom swing-signal mixer: pick factors, preview score vs built-in mix, save profiles.
 * Informational only — not financial advice.
 */
export function SignalsCustomPanel({
  profiles,
  selectedId,
  draft,
  items,
  loading = false,
  saving = false,
  scanning = false,
  saveError,
  onSelectProfile,
  onDraft,
  onSave,
  onSaveAs,
  onDelete,
  onApply,
  onReset,
  onOpen,
}: SignalsCustomPanelProps) {
  const { t } = useTranslation(['signals', 'common']);
  const [name, setName] = useState('');
  const preview = useMemo(() => previewCustomMix(items, draft), [items, draft]);
  const total = customWeightTotal(draft);
  const ok = customWeightsValid(draft);
  const huntOn = draftNeedsHunt(draft);
  const huntNeedsReload = huntOn && preview.some((row) => row.huntMissing);
  const selected = profiles.find((p) => p.id === selectedId);
  const builtIn = selectedId === CUSTOM_SIGNAL_BUILT_IN_ID;

  const options = [
    { value: CUSTOM_SIGNAL_BUILT_IN_ID, label: t('signals:custom.builtIn') },
    ...profiles.map((p) => ({ value: p.id ?? '', label: p.name ?? p.id })),
  ];

  return (
    <Layout>
      <Mixer>
        <Text variant="caption" color="secondary">
          {t('signals:custom.hint')}
        </Text>
        <Select
          value={selectedId}
          options={options}
          onChange={(id) => {
            setName('');
            onSelectProfile(id);
          }}
          aria-label={t('signals:custom.profile')}
        />
        <Input
          value={name || (builtIn ? '' : (selected?.name ?? ''))}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('signals:custom.namePlaceholder')}
          aria-label={t('signals:custom.name')}
        />
        <FactorList>
          {CUSTOM_SIGNAL_FACTORS.map((factor) => {
            const row = draft.find((d) => d.id === factor.id) ?? { id: factor.id, enabled: false, pct: 0 };
            return (
              <FactorRow key={factor.id}>
                <Switch
                  size="small"
                  checked={row.enabled}
                  onChange={(on) =>
                    onDraft(
                      draft.map((d) =>
                        d.id === factor.id
                          ? { ...d, enabled: on, pct: on ? d.pct || factor.defaultPct || 10 : 0 }
                          : d,
                      ),
                    )
                  }
                />
                <Text variant="bodySm" color="primary">
                  {t(`signals:custom.factors.${factor.id}`)}
                </Text>
                <InputNumber
                  min={0}
                  max={100}
                  step={1}
                  value={row.enabled ? row.pct : 0}
                  disabled={!row.enabled}
                  onChange={(value) =>
                    onDraft(draft.map((d) => (d.id === factor.id ? { ...d, pct: Number(value) || 0 } : d)))
                  }
                  aria-label={t(`signals:custom.factors.${factor.id}`)}
                />
              </FactorRow>
            );
          })}
        </FactorList>
        <Text variant="bodySm" color={ok ? 'secondary' : 'error'} data-testid="signals-custom-total">
          {t('signals:custom.total', { total: Math.round(total * 10) / 10 })}
          {!ok ? ` — ${t('signals:custom.needHundred')}` : ''}
        </Text>
        {huntNeedsReload ? (
          <Alert type="info" showIcon message={t('signals:custom.huntReload')} />
        ) : null}
        {saveError ? (
          <Alert type="error" showIcon message={rtkErrorMessage(saveError, { resource: t('signals:resource') })} />
        ) : null}
        <ActionRow>
          <Button type="primary" disabled={!ok} loading={scanning} onClick={onApply}>
            {t('signals:custom.apply')}
          </Button>
          <Button
            disabled={!ok || !name.trim()}
            loading={saving}
            onClick={() => onSaveAs(name.trim())}
          >
            {t('signals:custom.saveAs')}
          </Button>
          <Button
            disabled={builtIn || !ok}
            loading={saving}
            onClick={() => onSave((name || selected?.name || '').trim())}
          >
            {t('signals:custom.save')}
          </Button>
          <Button onClick={onReset}>{t('signals:custom.reset')}</Button>
          <Button danger disabled={builtIn} onClick={onDelete}>
            {t('signals:custom.delete')}
          </Button>
        </ActionRow>
      </Mixer>
      <Results>
        <Text variant="caption" color="secondary">
          {t('signals:custom.previewHint')}
        </Text>
        {loading && preview.length === 0 ? (
          <DeskEmpty title={t('signals:custom.loading')} />
        ) : preview.length === 0 ? (
          <DeskEmpty title={t('signals:custom.empty')} />
        ) : (
          preview.map((row) => (
            <Card key={`${row.exchange}|${row.symbol}`}>
              <CardTop>
                <div>
                  <Text variant="label" mono color="primary">
                    {formatSymbolDisplay(row.symbol)}
                  </Text>
                  <BrandTag variant="exchange">{row.exchange}</BrandTag>
                </div>
                <BrandTag variant={gradeVariant(row.grade)}>
                  {t('signals:custom.grade', { grade: row.grade, score: row.appliedScore })}
                </BrandTag>
              </CardTop>
              <Text variant="bodySm" color="secondary">
                {t('signals:custom.vsDefault', {
                  applied: row.appliedScore,
                  def: row.defaultScore,
                  delta: row.delta > 0 ? `+${row.delta}` : String(row.delta),
                })}
                {row.price ? ` · ${formatPrice(row.price)}` : ''}
              </Text>
              {row.largestChange ? (
                <Text variant="caption" color="secondary">
                  {t('signals:custom.largestChange', {
                    factor: t(`signals:custom.factors.${row.largestChange.id}`),
                    delta: row.largestChange.deltaEffect > 0
                      ? `+${row.largestChange.deltaEffect}`
                      : String(row.largestChange.deltaEffect),
                  })}
                </Text>
              ) : null}
              <FactorBreakdown>
                {row.factors
                  .filter((f) => f.appliedPct > 0 || f.defaultPct > 0)
                  .map((f) => (
                    <FactorLine key={f.id}>
                      <Text variant="caption" color="secondary">
                        {t(`signals:custom.factors.${f.id}`)}
                        {f.status === 'missing' ? ` (${t('signals:custom.missing')})` : ''}
                      </Text>
                      <Text variant="caption" mono color="secondary">
                        {f.appliedPct}%
                      </Text>
                      <Text variant="caption" mono color={f.appliedEffect >= 0 ? 'success' : 'error'}>
                        {f.appliedEffect > 0 ? `+${f.appliedEffect}` : f.appliedEffect}
                      </Text>
                    </FactorLine>
                  ))}
              </FactorBreakdown>
              <Button size="small" onClick={() => onOpen(row.exchange, row.symbol)}>
                {t('signals:openChart')}
              </Button>
            </Card>
          ))
        )}
      </Results>
    </Layout>
  );
}
