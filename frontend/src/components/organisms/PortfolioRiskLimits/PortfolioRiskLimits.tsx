import { useState } from 'react';
import { Button, InputNumber } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { useGetPortfolioRiskLimitsQuery, useSetPortfolioRiskLimitsMutation } from '@/libs/api';
import { exposureLine } from './helpers';
import type { PortfolioRiskLimitsProps } from './PortfolioRiskLimits.types';

export function PortfolioRiskLimits({ portfolioId, disabled }: PortfolioRiskLimitsProps) {
  const { t } = useTranslation('portfolio');
  const query = useGetPortfolioRiskLimitsQuery(portfolioId ? { portfolioId } : undefined, { skip: disabled });
  const [save, state] = useSetPortfolioRiskLimitsMutation();
  const saved = query.data?.limits?.maxExposurePct;
  const [draft, setDraft] = useState<number | null>(saved ?? null);
  const [touched, setTouched] = useState(false);
  const value = touched ? draft : (saved ?? null);
  const line = exposureLine(query.data?.status?.exposurePct, saved);

  return (
    <section data-testid="portfolio-exposure">
      <Text variant="h4" color="primary">
        {t('risk.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('risk.hint')}
      </Text>
      {line ? (
        <Text variant="numeric" color={query.data?.status?.exposureLimitHit ? 'error' : 'primary'}>
          {t('risk.now', { line })}
        </Text>
      ) : null}
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 8 }}>
        <InputNumber
          min={0.01}
          max={1000}
          value={value ?? undefined}
          disabled={disabled || state.isLoading}
          aria-label={t('risk.limit')}
          onChange={(n) => {
            setTouched(true);
            setDraft(typeof n === 'number' ? n : null);
          }}
        />
        <Button
          type="primary"
          disabled={disabled}
          loading={state.isLoading}
          onClick={() => {
            const limits = query.data?.limits;
            void save({
              portfolioId,
              maxDailyLossPct: limits?.maxDailyLossPct ?? null,
              maxAssetWeightPct: limits?.maxAssetWeightPct ?? null,
              maxExposurePct: value,
            });
          }}
        >
          {t('risk.save')}
        </Button>
      </div>
    </section>
  );
}
