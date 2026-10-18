import { useState } from 'react';
import { Button, InputNumber, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { useGetEmergencyStopQuery, useSetEmergencyStopMutation } from '@/libs/api';
import type { EmergencyKind } from './constants';
import { EMERGENCY_KINDS } from './constants';
import { lossLine } from './helpers';
import type { EmergencyStopProps } from './EmergencyStop.types';

function isKind(kind: string | undefined): kind is EmergencyKind {
  return EMERGENCY_KINDS.some((item) => item === kind);
}

/**
 * Arms a paper-book stop that flattens when today's trading loss passes a percent.
 * The page owns the book id. This reads and saves the switch.
 */
export function EmergencyStop({ portfolioId, disabled }: EmergencyStopProps) {
  const { t } = useTranslation('portfolio');
  const query = useGetEmergencyStopQuery(portfolioId ? { portfolioId } : undefined, {
    skip: disabled,
    pollingInterval: disabled ? 0 : 15_000,
    refetchOnFocus: true,
  });
  const [save, state] = useSetEmergencyStopMutation();
  const savedOn = query.data?.enabled ?? false;
  const savedPct = query.data?.lossPct;
  const [enabled, setEnabled] = useState(savedOn);
  const [loss, setLoss] = useState<number | null>(savedPct && savedPct > 0 ? savedPct : null);
  const [touched, setTouched] = useState(false);
  const on = touched ? enabled : savedOn;
  const pct = touched ? loss : savedPct && savedPct > 0 ? savedPct : null;
  const actions = query.data?.run?.actions ?? [];

  return (
    <section data-testid="emergency-stop">
      <Text variant="h4" color="primary">
        {t('emergency.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('emergency.hint')}
      </Text>
      <Text variant="numeric" color={query.data?.hit ? 'error' : 'primary'}>
        {t('emergency.now', { line: lossLine(query.data?.dailyPnlPct) })}
      </Text>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 8, flexWrap: 'wrap' }}>
        <Switch
          checked={on}
          disabled={disabled || state.isLoading}
          aria-label={t('emergency.switch')}
          onChange={(next) => {
            setTouched(true);
            setEnabled(next);
          }}
        />
        <InputNumber
          min={0.01}
          max={100}
          value={pct ?? undefined}
          disabled={disabled || state.isLoading}
          aria-label={t('emergency.loss')}
          onChange={(n) => {
            setTouched(true);
            setLoss(typeof n === 'number' ? n : null);
          }}
        />
        <Button
          type="primary"
          disabled={disabled}
          loading={state.isLoading}
          onClick={() => {
            void save({ portfolioId, enabled: on, lossPct: pct ?? 0 });
          }}
        >
          {t('emergency.save')}
        </Button>
      </div>
      {actions.length > 0 ? (
        <div data-testid="emergency-actions">
          {actions.map((action) => {
            const kind = isKind(action.kind) ? action.kind : 'cancel_pending';
            const name = action.symbol || action.targetId || '';
            return (
              <Text key={`${action.kind}-${action.targetId}`} variant="caption">
                {t(`emergency.kind.${kind}`, { symbol: name })}
                {' · '}
                {action.status === 'failed'
                  ? t('emergency.failed', { detail: action.detail || '' })
                  : t('emergency.ok')}
              </Text>
            );
          })}
        </div>
      ) : null}
    </section>
  );
}
