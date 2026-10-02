import { useTranslation } from 'react-i18next';
import { CoinInsightPanel, CoinInsightStack } from '@/components/organisms/CoinInsightPanel';
import { VenueDivergenceBoard } from '@/components/organisms/VenueDivergenceBoard';
import {
  rtkErrorMessage,
  useGetMarketCorrelationQuery,
  useGetMarketLevelsQuery,
  useGetMarketPositioningQuery,
  useGetMarketSqueezeRiskQuery,
  useGetMarketVenueDivergenceQuery,
  useGetMarketVolatilityQuery,
} from '@/libs/api';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import { reportSummary } from './CoinFlowTab.helpers';
import type { CoinStructureTabProps } from './CoinStructureTab.types';

/** Per-coin structure: correlation, vol, levels, squeeze, positioning, venue gap. */
export function CoinStructureTab({ symbol, exchange }: CoinStructureTabProps) {
  const { t } = useTranslation('detail');
  const arg = { symbol, exchange };
  const futuresArg = { symbol, exchange: 'all' as const };
  const err = (e: unknown) => rtkErrorMessage(e, { resource: t('resource.tape') });

  const corr = useGetMarketCorrelationQuery(arg);
  const vol = useGetMarketVolatilityQuery(arg);
  const levels = useGetMarketLevelsQuery(arg);
  const squeeze = useGetMarketSqueezeRiskQuery(futuresArg);
  const pos = useGetMarketPositioningQuery(futuresArg);
  const div = useGetMarketVenueDivergenceQuery(futuresArg);

  const rows = [
    { key: 'corr', title: t('structure.correlation'), q: corr },
    { key: 'vol', title: t('structure.volatility'), q: vol },
    { key: 'levels', title: t('structure.levels'), q: levels },
    { key: 'squeeze', title: t('structure.squeeze'), q: squeeze },
    { key: 'pos', title: t('structure.positioning'), q: pos },
    { key: 'div', title: t('structure.divergence'), q: div },
  ];

  const divReport = rtkCurrent(div);
  return (
    <CoinInsightStack>
      {rows.map((row) => (
        <CoinInsightPanel
          key={row.key}
          title={row.title}
          summary={row.key === 'div' ? (divReport?.summary ?? '') : reportSummary(rtkCurrent(row.q))}
          extra={row.key === 'div' ? <VenueDivergenceBoard report={divReport} /> : undefined}
          error={row.q.isError ? err(row.q.error) : null}
          isLoading={rtkCurrentPending(row.q)}
        />
      ))}
    </CoinInsightStack>
  );
}
