import {
  PositionSituation,
  isCashEquity,
  positionLotReads,
} from '@/components/molecules/PositionSituation';
import {
  useGetMarketSituationQuery,
  useGetPortfolioSituationChangeQuery,
  type SpotPosition,
} from '@/libs/api';
import { POSITION_SITUATION_POLL_MS } from './constants';

export function PositionRead({ row, portfolioId }: { row: SpotPosition; portfolioId?: string }) {
  const equity = isCashEquity(row.exchange);
  const hasEntry = Boolean(row.entrySituation) || (row.openReads?.length ?? 0) > 0;
  const change = useGetPortfolioSituationChangeQuery(
    { symbol: row.symbol ?? '', exchange: row.exchange, portfolioId },
    {
      skip: equity || !row.symbol || !hasEntry,
      pollingInterval: POSITION_SITUATION_POLL_MS,
      refetchOnFocus: true,
    },
  );
  const changeFailed =
    hasEntry && (change.isError || (change.data != null && change.data.liveOk === false));
  const query = useGetMarketSituationQuery(
    { symbol: row.symbol ?? '' },
    {
      skip: equity || !row.symbol || (hasEntry && !changeFailed),
      pollingInterval: POSITION_SITUATION_POLL_MS,
      refetchOnFocus: true,
    },
  );
  if (equity) return null;
  const fromChange = change.data?.liveOk ? change.data.live : undefined;
  const now = fromChange ?? query.data;
  const nowLoading =
    now == null && (hasEntry ? change.isLoading && !changeFailed : query.isLoading);
  return (
    <PositionSituation
      entry={row.entrySituation}
      lots={positionLotReads(row, change.data)}
      now={now}
      nowLoading={nowLoading}
      comparison={change.data?.liveOk ? change.data.comparison : undefined}
    />
  );
}
