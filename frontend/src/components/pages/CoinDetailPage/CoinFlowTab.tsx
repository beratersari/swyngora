import { useTranslation } from 'react-i18next';
import { CoinInsightPanel, CoinInsightStack } from '@/components/organisms/CoinInsightPanel';
import { VolumeProfileLadder } from '@/components/organisms/VolumeProfileLadder';
import { PositionSidePanel } from '@/components/organisms/PositionSidePanel';
import {
  rtkErrorMessage,
  useGetMarketAbsorptionQuery,
  useGetMarketBookLiquidityFlowQuery,
  useGetMarketFlowAtPricesQuery,
  useGetMarketIcebergsQuery,
  useGetMarketLiquiditySweepsQuery,
  useGetMarketPerpDepthQuery,
  useGetMarketTakerFlowQuery,
  useGetMarketNetPositionQuery,
  useGetMarketVolumeProfileQuery,
  useGetMarketVolumeSurgeQuery,
  useGetMarketVWAPQuery,
  useGetMarketWhalesQuery,
} from '@/libs/api';
import { rtkCurrent, rtkCurrentPending } from '@/libs/utils';
import { perpStats, reportSummary, surgeStats } from './CoinFlowTab.helpers';
import { WhaleFollowList } from './WhaleFollowList';
import type { CoinFlowTabProps } from './CoinFlowTab.types';

/** Per-coin prints, book, and volume — RTK lives here (page layer), not in organisms. */
export function CoinFlowTab({ symbol, exchange }: CoinFlowTabProps) {
  const { t } = useTranslation('detail');
  const arg = { symbol, exchange };
  const tapeArg = { symbol, exchange: 'all' as const };
  const err = (e: unknown) => rtkErrorMessage(e, { resource: t('resource.tape') });

  const perp = useGetMarketPerpDepthQuery(tapeArg);
  const surge = useGetMarketVolumeSurgeQuery(tapeArg);
  const taker = useGetMarketTakerFlowQuery(tapeArg);
  const netPosition = useGetMarketNetPositionQuery(tapeArg);
  const flow = useGetMarketFlowAtPricesQuery({ ...tapeArg, window: '15m' });
  const vwap = useGetMarketVWAPQuery({ ...tapeArg, window: '24h' });
  const profile = useGetMarketVolumeProfileQuery(tapeArg);
  const abs = useGetMarketAbsorptionQuery(arg);
  const sweeps = useGetMarketLiquiditySweepsQuery(arg);
  const whales = useGetMarketWhalesQuery(arg);
  const ice = useGetMarketIcebergsQuery(arg);
  const liq = useGetMarketBookLiquidityFlowQuery({ ...arg, window: '15m' });

  const rows = [
    { key: 'perp', title: t('flow.perpDepth'), q: perp, stats: perpStats(rtkCurrent(perp)) },
    { key: 'surge', title: t('flow.surge'), q: surge, stats: surgeStats(rtkCurrent(surge)) },
    { key: 'taker', title: t('flow.taker'), q: taker, stats: [] },
    { key: 'prices', title: t('flow.prices'), q: flow, stats: [] },
    { key: 'vwap', title: t('flow.vwap'), q: vwap, stats: [] },
    { key: 'profile', title: t('flow.profile'), q: profile, stats: [] },
    { key: 'abs', title: t('flow.absorption'), q: abs, stats: [] },
    { key: 'sweeps', title: t('flow.sweeps'), q: sweeps, stats: [] },
    { key: 'whales', title: t('flow.whales'), q: whales, stats: [] },
    { key: 'ice', title: t('flow.icebergs'), q: ice, stats: [] },
    { key: 'book', title: t('flow.bookFlow'), q: liq, stats: [] },
  ];

  return (
    <CoinInsightStack>
      <PositionSidePanel
        data={rtkCurrent(netPosition)}
        error={netPosition.isError ? err(netPosition.error) : null}
        isLoading={rtkCurrentPending(netPosition)}
      />
      {rows.map((row) => (
        <CoinInsightPanel
          key={row.key}
          title={row.title}
          summary={reportSummary(rtkCurrent(row.q))}
          stats={row.stats}
          extra={
            row.key === 'whales' ? (
              <WhaleFollowList events={rtkCurrent(whales)?.events} />
            ) : row.key === 'profile' ? (
              <VolumeProfileLadder report={rtkCurrent(profile)} />
            ) : null
          }
          error={row.q.isError ? err(row.q.error) : null}
          isLoading={rtkCurrentPending(row.q)}
        />
      ))}
    </CoinInsightStack>
  );
}
